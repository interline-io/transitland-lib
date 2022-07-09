package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

type Translation struct {
	// "TableNameValue" because TableName is a required interface method
	TableNameValue String `db:"table_name" csv:"table_name"`
	FieldName      String
	Language       Language
	Translation    String
	RecordID       String
	RecordSubID    String
	FieldValue     String
	BaseEntity
}

func (ent *Translation) Filename() string {
	return "translations.txt"
}

func (ent *Translation) TableName() string {
	return "gtfs_translations"
}

// Errors for this Entity.
func (ent *Translation) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("table_name", ent.TableNameValue.String)...)
	errs = append(errs, enum.CheckPresent("field_name", ent.FieldName.String)...)
	errs = enum.CheckError(errs, enum.CheckFieldPresentError("language", &ent.Language))
	errs = append(errs, enum.CheckPresent("translation", ent.Translation.String)...)
	errs = append(errs, enum.CheckInArray("table_name", ent.TableNameValue.String, "agency", "routes", "stops", "trips", "stop_times", "pathways", "levels", "feed_info", "attributions")...)
	// RecordID
	if ent.RecordID.String == "" {
		// Check this way because it's both forbidden and required when table is feed_info
		if ent.TableNameValue.String == "feed_info" {
			// empty ok when TableNameValue is feed_info
		} else if ent.FieldValue.String == "" {
			// required when FieldValue is also empty
			errs = append(errs, causes.NewConditionallyRequiredFieldError("record_id"))
		}
	} else {
		if ent.TableNameValue.String == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_id", "forbidden if table_name is 'feed_info'"))
		}
		// exclusive with FieldValue checked below
	}
	// RecordSubID
	if ent.RecordSubID.String == "" {
		if ent.TableNameValue.String == "stop_times" && ent.RecordID.String != "" {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("record_sub_id"))
		}
	} else {
		if ent.RecordID.String == "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", "forbidden if record_id is empty"))
		}
		if ent.TableNameValue.String == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", "forbidden if table_name is 'feed_info'"))
		}
		// exclusive with FieldValue checked below
	}
	// FieldValue
	if ent.FieldValue.String != "" {
		if ent.TableNameValue.String == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", "forbidden if table_name is 'feed_info'"))
		}
		if ent.RecordID.String != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", "forbidden if record_id is present"))
		}
	}
	return errs
}
