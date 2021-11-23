package soft_delete

import (
	"errors"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type IsDeleted bool

func (i *IsDeleted) Scan(value interface{}) error {
	if i == nil {
		return errors.New("value must be initialized (*IsDeleted)(nil)")
	}

	*i = false
	if value == true {
		*i = true
	}

	return nil
}

func (IsDeleted) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{SoftDeleteQueryClause{Field: f}}
}

type SoftDeleteQueryClause struct {
	Field *schema.Field
}

func (sd SoftDeleteQueryClause) Name() string {
	return ""
}

func (sd SoftDeleteQueryClause) Build(clause.Builder) {
}

func (sd SoftDeleteQueryClause) MergeClause(*clause.Clause) {
}

func (sd SoftDeleteQueryClause) ModifyStatement(stmt *gorm.Statement) {
	if _, ok := stmt.Clauses["soft_delete_enabled"]; !ok {
		if c, ok := stmt.Clauses["WHERE"]; ok {
			if where, ok := c.Expression.(clause.Where); ok && len(where.Exprs) > 1 {
				for _, expr := range where.Exprs {
					if orCond, ok := expr.(clause.OrConditions); ok && len(orCond.Exprs) == 1 {
						where.Exprs = []clause.Expression{clause.And(where.Exprs...)}
						c.Expression = where
						stmt.Clauses["WHERE"] = c
						break
					}
				}
			}
		}

		if sd.Field.DefaultValue == "null" {
			stmt.AddClause(clause.Where{Exprs: []clause.Expression{
				clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: sd.Field.DBName}, Value: nil},
			}})
		} else {
			stmt.AddClause(clause.Where{Exprs: []clause.Expression{
				clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: sd.Field.DBName}, Value: false},
			}})
		}
		stmt.Clauses["soft_delete_enabled"] = clause.Clause{}
	}
}

func (IsDeleted) DeleteClauses(f *schema.Field) []clause.Interface {
	softDeleteClause := SoftDeleteDeleteClause{
		Field: f,
	}

	deletedAtFieldName, ok := f.TagSettings["DELETEDATFIELD"]
	if ok && len(deletedAtFieldName) > 0 { // DeletedAtField
		softDeleteClause.DeleteAtField = f.Schema.LookUpField(deletedAtFieldName)

	}

	return []clause.Interface{softDeleteClause}
}

func (IsDeleted) UpdateClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{SoftDeleteUpdateClause{Field: f}}
}

type SoftDeleteUpdateClause struct {
	Field *schema.Field
}

func (sd SoftDeleteUpdateClause) Name() string {
	return ""
}

func (sd SoftDeleteUpdateClause) Build(clause.Builder) {
}

func (sd SoftDeleteUpdateClause) MergeClause(*clause.Clause) {
}

func (sd SoftDeleteUpdateClause) ModifyStatement(stmt *gorm.Statement) {
	if stmt.SQL.String() == "" {
		if _, ok := stmt.Clauses["WHERE"]; stmt.DB.AllowGlobalUpdate || ok {
			SoftDeleteQueryClause(sd).ModifyStatement(stmt)
		}
	}
}

type SoftDeleteDeleteClause struct {
	Field         *schema.Field
	DeleteAtField *schema.Field
}

func (sd SoftDeleteDeleteClause) Name() string {
	return ""
}

func (sd SoftDeleteDeleteClause) Build(clause.Builder) {
}

func (sd SoftDeleteDeleteClause) MergeClause(*clause.Clause) {
}

func (sd SoftDeleteDeleteClause) ModifyStatement(stmt *gorm.Statement) {
	if stmt.SQL.String() == "" {
		curTime := stmt.DB.NowFunc()

		set := clause.Set{{Column: clause.Column{Name: sd.Field.DBName}, Value: true}}
		stmt.SetColumn(sd.Field.DBName, true, true)

		if sd.DeleteAtField != nil {
			set = append(set, clause.Assignment{Column: clause.Column{Name: sd.DeleteAtField.DBName}, Value: curTime})
			stmt.SetColumn(sd.DeleteAtField.DBName, curTime, true)
		}

		stmt.AddClause(set)

		if stmt.Schema != nil {
			_, queryValues := schema.GetIdentityFieldValuesMap(stmt.ReflectValue, stmt.Schema.PrimaryFields)
			column, values := schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

			if len(values) > 0 {
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
			}

			if stmt.ReflectValue.CanAddr() && stmt.Dest != stmt.Model && stmt.Model != nil {
				_, queryValues = schema.GetIdentityFieldValuesMap(reflect.ValueOf(stmt.Model), stmt.Schema.PrimaryFields)
				column, values = schema.ToQueryValues(stmt.Table, stmt.Schema.PrimaryFieldDBNames, queryValues)

				if len(values) > 0 {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}
			}
		}

		if _, ok := stmt.Clauses["WHERE"]; !stmt.DB.AllowGlobalUpdate && !ok {
			stmt.DB.AddError(gorm.ErrMissingWhereClause)
		} else {
			SoftDeleteQueryClause{Field: sd.Field}.ModifyStatement(stmt)
		}

		stmt.AddClauseIfNotExists(clause.Update{})
		stmt.Build(stmt.DB.Callback().Update().Clauses...)
	}
}
