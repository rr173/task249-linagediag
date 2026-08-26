package transform

import (
	"sort"
	"strings"
)

// RenameCandidate 表示对「缺失源列」推断出的可能重命名候选。
type RenameCandidate struct {
	Table      string  `json:"table"`
	OldName    string  `json:"old_name"`
	NewName    string  `json:"new_name"`
	Confidence float64 `json:"confidence"`
}

// InferRenameCandidates 对缺失的源列，在同一表的最新版本里查找「词干相似」的列作为重命名候选。
// 用于诊断「上游列重命名后下游派生列未更新」这一类截断。
func (svc *Service) InferRenameCandidates(batchID int64, table, oldColumn string) ([]RenameCandidate, error) {
	tbl, err := svc.store.LatestTableVersion(batchID, table)
	if err != nil {
		return nil, err
	}
	cols, err := svc.store.ListColumns(tbl.ID)
	if err != nil {
		return nil, err
	}
	var out []RenameCandidate
	for _, c := range cols {
		if c.Name == oldColumn {
			continue
		}
		conf := renameConfidence(oldColumn, c.Name)
		if conf > 0 {
			out = append(out, RenameCandidate{
				Table:      table,
				OldName:    oldColumn,
				NewName:    c.Name,
				Confidence: conf,
			})
		}
	}
	// 候选按置信度降序排列，最相似的重命名优先（diagnosis/locate、
	// service/lineage 的自动修订均取 [0]）。同分时按 NewName 字典序稳定 tie-break，
	// 保证上游列同时可能改成两个相似名称时给出确定且最相近的优先候选。
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].NewName < out[j].NewName
	})
	return out, nil
}

// renameConfidence 计算 old→cand 的重命名相似度（0 表示不相似）。
// 规则：显式版本后缀(_v2/_new)与词干前缀匹配给高分；其余按前缀重叠比例给分。
func renameConfidence(old, cand string) float64 {
	if old == "" || cand == "" || old == cand {
		return 0
	}
	o, c := strings.ToLower(old), strings.ToLower(cand)
	if c == o+"_v2" || c == o+"_v3" || c == o+"_new" || c == o+"_updated" {
		return 0.95
	}
	if strings.HasPrefix(c, o+"_") || strings.HasPrefix(o, c+"_") {
		return 0.85
	}
	short := o
	if len(c) < len(o) {
		short = c
	}
	prefix := 0
	for i := 0; i < len(short); i++ {
		if o[i] == c[i] {
			prefix++
		} else {
			break
		}
	}
	ratio := float64(prefix) / float64(len(short))
	if ratio >= 0.6 {
		return ratio * 0.7
	}
	return 0
}
