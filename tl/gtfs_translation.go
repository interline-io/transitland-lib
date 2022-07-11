package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type Translation struct {
	// "TableNameValue" because TableName is a required interface method
	TableNameValue String `db:"table_name" csv:"table_name"`
	FieldName      String
	Language       String
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
	errs = append(errs, tt.CheckPresent("table_name", ent.TableNameValue.Val)...)
	errs = append(errs, tt.CheckPresent("field_name", ent.FieldName.Val)...)
	errs = append(errs, tt.CheckPresent("language", ent.Language.Val)...)
	errs = append(errs, tt.CheckLanguage("language", ent.Language.Val)...)
	errs = append(errs, tt.CheckPresent("translation", ent.Translation.Val)...)
	errs = append(errs, tt.CheckInArray("table_name", ent.TableNameValue.Val, "agency", "routes", "stops", "trips", "stop_times", "pathways", "levels", "feed_info", "attributions")...)
	// RecordID
	if ent.RecordID.Val == "" {
		// Check this way because it's both forbidden and required when table is feed_info
		if ent.TableNameValue.Val == "feed_info" {
			// empty ok when TableNameValue is feed_info
		} else if ent.FieldValue.Val == "" {
			// required when FieldValue is also empty
			errs = append(errs, causes.NewConditionallyRequiredFieldError("record_id"))
		}
	} else {
		if ent.TableNameValue.Val == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_id", "forbidden if table_name is 'feed_info'"))
		}
		// exclusive with FieldValue checked below
	}
	// RecordSubID
	if ent.RecordSubID.Val == "" {
		if ent.TableNameValue.Val == "stop_times" && ent.RecordID.Val != "" {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("record_sub_id"))
		}
	} else {
		if ent.RecordID.Val == "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", "forbidden if record_id is empty"))
		}
		if ent.TableNameValue.Val == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", "forbidden if table_name is 'feed_info'"))
		}
		// exclusive with FieldValue checked below
	}
	// FieldValue
	if ent.FieldValue.Val != "" {
		if ent.TableNameValue.Val == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", "forbidden if table_name is 'feed_info'"))
		}
		if ent.RecordID.Val != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", "forbidden if record_id is present"))
		}
	}
	return errs
}
