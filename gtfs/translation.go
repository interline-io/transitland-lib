package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

type Translation struct {
	// "TableNameValue" because TableName is a required interface method
	TableNameValue tt.String   `db:"table_name" csv:"table_name,required"`
	FieldName      tt.String   `csv:",required"`
	Language       tt.Language `csv:",required"`
	Translation    tt.String   `csv:",required"`
	RecordID       tt.String
	RecordSubID    tt.String
	FieldValue     tt.String
	tt.BaseEntity
}

func (ent *Translation) Filename() string {
	return "translations.txt"
}

func (ent *Translation) TableName() string {
	return "gtfs_translations"
}

// Errors for this Entity.
func (ent *Translation) ConditionalErrors() (errs []error) {
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
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_id", ent.TableNameValue.Val, "forbidden if table_name is 'feed_info'"))
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
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", ent.RecordID.Val, "forbidden if record_id is empty"))
		}
		if ent.TableNameValue.Val == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", ent.TableNameValue.Val, "forbidden if table_name is 'feed_info'"))
		}
		// exclusive with FieldValue checked below
	}
	// FieldValue
	if ent.FieldValue.Val != "" {
		if ent.TableNameValue.Val == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", ent.TableNameValue.Val, "forbidden if table_name is 'feed_info'"))
		}
		if ent.RecordID.Val != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", ent.RecordID.Val, "forbidden if record_id is present"))
		}
	}
	return errs
}
