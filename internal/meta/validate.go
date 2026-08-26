package meta

import (
	"fmt"
	"strings"

	"task249-linagediag/internal/model"
)

// 类型族：用于判定「源列 → 目标列」的类型相容性，避免跨类型错误派生。
var numericTypes = map[string]bool{
	"int": true, "integer": true, "bigint": true, "smallint": true, "tinyint": true,
	"decimal": true, "numeric": true, "float": true, "double": true, "real": true,
	"number": true,
}

var stringTypes = map[string]bool{
	"string": true, "text": true, "varchar": true, "char": true, "nvarchar": true,
}

var timeTypes = map[string]bool{
	"timestamp": true, "datetime": true, "date": true, "time": true,
}

// typeFamily 返回类型的归一类族；未知类型归入 "other"（与其它 other 相容）。
func typeFamily(dt string) string {
	key := strings.ToLower(strings.TrimSpace(dt))
	if numericTypes[key] {
		return "numeric"
	}
	if stringTypes[key] {
		return "string"
	}
	if timeTypes[key] {
		return "time"
	}
	if key == "bool" || key == "boolean" {
		return "bool"
	}
	return "other"
}

// CompatibleTypes 判断源类型能否派生到目标类型（同族相容，other 仅与 other 相容）。
func CompatibleTypes(source, target string) bool {
	fs, ft := typeFamily(source), typeFamily(target)
	if fs == "other" || ft == "other" {
		return fs == ft
	}
	return fs == ft
}

// ValidateTransformTypes 在构图前校验一条变换的源/目标列类型是否相容。
func ValidateTransformTypes(srcType, tgtType string) error {
	if !CompatibleTypes(srcType, tgtType) {
		return fmt.Errorf("%w: %s -> %s", model.ErrTypeIncompatible, srcType, tgtType)
	}
	return nil
}
