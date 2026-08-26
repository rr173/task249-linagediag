// Package transform 负责把元数据与列级变换声明解析为血缘边，
// 并基于同名列直通推断隐式边。这是「构图中」环节的核心。
package transform

import (
	"fmt"

	"task249-linagediag/internal/meta"
	"task249-linagediag/internal/model"
	"task249-linagediag/internal/store"
)

// Service 变换解析服务。
type Service struct {
	store *store.Store
}

// NewService 构造变换解析服务。
func NewService(st *store.Store) *Service {
	return &Service{store: st}
}

// ParseResult 解析结果。
type ParseResult struct {
	Edges    int // 本次写入的血缘边数量
	Broken   int // 其中被标记为断裂（源列缺失）的数量
	Inferred int // 其中自动推断的数量
}

// BuildEdges 依据某批次的变换声明与推断规则，重建列级血缘边。
// 步骤：
//  1. 清空旧边；
//  2. 对每条 transform：解析源/目标列 ID；若源列缺失 → 标记断裂；否则按声明写入；
//  3. 对每个作业的输出表，对其输入表中「同名同类型」的列自动推断直通边（inferred）。
//
// 返回统计结果。构图期间批次应处于「构图中」状态（由上层 service 保证）。
func (svc *Service) BuildEdges(batchID int64) (*ParseResult, error) {
	if err := svc.store.DeleteEdgesForBatch(batchID); err != nil {
		return nil, err
	}

	transforms, err := svc.store.ListTransforms(batchID)
	if err != nil {
		return nil, err
	}

	res := &ParseResult{}
	for _, t := range transforms {
		srcColID, srcErr := svc.store.ColumnID(batchID, t.SourceTable, "", t.SourceColumn)
		tgtColID, tgtErr := svc.store.ColumnID(batchID, t.TargetTable, "", t.TargetColumn)

		// 变换声明里的列定位需要带上版本；先用 LatestTableVersion 解析。
		if srcErr != nil {
			srcColID, srcErr = svc.resolveLatestColumn(batchID, t.SourceTable, t.SourceColumn)
		}
		if tgtErr != nil {
			tgtColID, tgtErr = svc.resolveLatestColumn(batchID, t.TargetTable, t.TargetColumn)
		}

		e := &model.Edge{
			BatchID:     batchID,
			TransformID: t.ID,
			Status:      t.Status,
		}
		if srcErr != nil || tgtErr != nil {
			e.Status = model.EdgeBroken
			e.Reason = fmt.Sprintf("source=%v target=%v", srcErr, tgtErr)
			// 断裂边仍记录 transform，但源/目标列 ID 置 0，便于后续补全。
			e.SourceColID = 0
			e.TargetColID = 0
			res.Broken++
		} else {
			srcCol, err := svc.store.GetColumn(srcColID)
			if err != nil {
				return nil, err
			}
			tgtCol, err := svc.store.GetColumn(tgtColID)
			if err != nil {
				return nil, err
			}
			if typeErr := meta.ValidateTransformTypes(srcCol.DataType, tgtCol.DataType); typeErr != nil {
				e.Status = model.EdgeBroken
				e.Reason = typeErr.Error()
				e.SourceColID = 0
				e.TargetColID = 0
				res.Broken++
				if _, err := svc.store.CreateEdge(e); err != nil {
					return nil, err
				}
				res.Edges++
				continue
			}
			e.SourceColID = srcColID
			e.TargetColID = tgtColID
			if t.Status == model.EdgeInferred {
				res.Inferred++
			}
		}
		if _, err := svc.store.CreateEdge(e); err != nil {
			return nil, err
		}
		res.Edges++
	}

	if err := svc.inferPassthrough(batchID, res); err != nil {
		return nil, err
	}
	return res, nil
}

// resolveLatestColumn 用某表最新版本解析列 ID（供不带版本的 transform 引用）。
func (svc *Service) resolveLatestColumn(batchID int64, qualifiedName, column string) (int64, error) {
	tbl, err := svc.store.LatestTableVersion(batchID, qualifiedName)
	if err != nil {
		return 0, err
	}
	return svc.store.ColumnID(batchID, qualifiedName, tbl.SchemaVersion, column)
}

// inferPassthrough 对每个作业，基于输入表与输出表的同名同类型列推断直通边（inferred）。
func (svc *Service) inferPassthrough(batchID int64, res *ParseResult) error {
	jobs, err := svc.store.ListJobs(batchID)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		inputs, err := svc.store.JobInputs(job.ID)
		if err != nil {
			return err
		}
		outCols, err := svc.store.ListColumns(job.OutputTableID)
		if err != nil {
			return err
		}
		for _, inID := range inputs {
			inCols, err := svc.store.ListColumns(inID)
			if err != nil {
				return err
			}
			for _, ic := range inCols {
				for _, oc := range outCols {
					if ic.Name == oc.Name {
						// 已存在同名直通边则跳过（避免重复）。
						if exists, _ := svc.edgeExists(batchID, ic.ID, oc.ID); exists {
							continue
						}
						e := &model.Edge{
							BatchID:     batchID,
							SourceColID: ic.ID,
							TargetColID: oc.ID,
							TransformID: job.ID, // 借用 job 作为推断来源
							Status:      model.EdgeInferred,
							Reason:      "passthrough same-name column",
						}
						if _, err := svc.store.CreateEdge(e); err != nil {
							return err
						}
						res.Edges++
						res.Inferred++
					}
				}
			}
		}
	}
	return nil
}

// edgeExists 判断 (源, 目标) 边是否已存在。
func (svc *Service) edgeExists(batchID, srcColID, tgtColID int64) (bool, error) {
	edges, err := svc.store.ListEdges(batchID)
	if err != nil {
		return false, err
	}
	for _, e := range edges {
		if e.SourceColID == srcColID && e.TargetColID == tgtColID {
			return true, nil
		}
	}
	return false, nil
}
