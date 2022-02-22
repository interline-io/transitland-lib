package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

type Translation struct {
	// "TableNameValue" because TableName is a required interface method
	TableNameValue OString `db:"table_name" csv:"table_name"`
	FieldName      OString
	Language       OString
	Translation    OString
	RecordID       OString
	RecordSubID    OString
	FieldValue     OString
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
	errs = append(errs, enum.CheckPresent("language", ent.Language.String)...)
	errs = append(errs, enum.CheckLanguage("language", ent.Language.String)...)
	errs = append(errs, enum.CheckPresent("translation", ent.Translation.String)...)
	errs = append(errs, enum.CheckInArray("table_name", ent.TableNameValue.String, "agency", "stops", "trips", "stop_times", "pathways", "levels", "feed_info", "attributions")...)
	// RecordID
	if ent.RecordID.String == "" {
		if ent.FieldValue.String == "" {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("record_id"))
		}
	} else {
		if ent.TableNameValue.String == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_id", "forbidden if table_name is 'feed_info'"))
		}
		if ent.FieldValue.String != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_id", "forbidden if field_value is present"))
		}
	}
	// RecordSubID
	if ent.RecordSubID.String == "" {
		if ent.TableNameValue.String == "stop_times" && ent.RecordID.String != "" {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("record_sub_id"))
		}
	} else {
		if ent.TableNameValue.String == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", "forbidden if table_name is 'feed_info'"))
		}
		if ent.FieldValue.String != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("record_sub_id", "forbidden if field_value is present"))
		}
	}
	// FieldValue
	if ent.FieldValue.String == "" {
		if ent.TableNameValue.String == "feed_info" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", "forbidden if table_name is 'feed_info'"))
		}
		if ent.RecordID.String != "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", "forbidden if record_id is present"))
		}
	} else {
		if ent.RecordID.String == "" {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("field_value", "required if record_id is empty"))
		}
	}
	return errs
}
