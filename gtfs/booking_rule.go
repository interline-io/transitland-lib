package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// BookingRule booking_rules.txt
type BookingRule struct {
	BookingRuleID          tt.String `csv:",required"`
	BookingType            tt.Int    `csv:",required" enum:"0,1,2"`
	PriorNoticeDurationMin tt.Int
	PriorNoticeDurationMax tt.Int
	PriorNoticeLastDay     tt.Int
	PriorNoticeLastTime    tt.Seconds
	PriorNoticeStartDay    tt.Int
	PriorNoticeStartTime   tt.Seconds
	PriorNoticeServiceID   tt.Key `target:"calendar.txt"`
	Message                tt.String
	PickupMessage          tt.String
	DropOffMessage         tt.String
	PhoneNumber            tt.String
	InfoURL                tt.Url
	BookingURL             tt.Url
	tt.BaseEntity
}

func (ent *BookingRule) EntityKey() string {
	return ent.BookingRuleID.Val
}

func (ent *BookingRule) EntityID() string {
	return entID(ent.ID, ent.BookingRuleID.Val)
}

func (ent *BookingRule) Filename() string {
	return "booking_rules.txt"
}

func (ent *BookingRule) TableName() string {
	return "gtfs_booking_rules"
}

// UpdateKeys updates Entity references.
func (ent *BookingRule) UpdateKeys(emap *tt.EntityMap) error {
	return tt.TrySetField(emap.UpdateKey(&ent.PriorNoticeServiceID, "calendar.txt"), "prior_notice_service_id")
}

// ConditionalErrors for this Entity.
func (ent *BookingRule) ConditionalErrors() (errs []error) {
	bookingType := ent.BookingType.Val

	// booking_type=1: prior_notice_duration_min is required
	if bookingType == 1 {
		if !ent.PriorNoticeDurationMin.Valid {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("prior_notice_duration_min"))
		}
		// prior_notice_duration_max is optional for booking_type=1
	} else {
		// prior_notice_duration_min is forbidden for booking_type=0 and 2
		if ent.PriorNoticeDurationMin.Valid {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("prior_notice_duration_min", ent.PriorNoticeDurationMin.String(), "prior_notice_duration_min is forbidden for booking_type!=1"))
		}
	}

	// booking_type=2: prior_notice_last_day is required
	if bookingType == 2 {
		if !ent.PriorNoticeLastDay.Valid {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("prior_notice_last_day"))
		}
		// prior_notice_duration_max is forbidden for booking_type=2
		if ent.PriorNoticeDurationMax.Valid {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("prior_notice_duration_max", ent.PriorNoticeDurationMax.String(), "prior_notice_duration_max is forbidden for booking_type=2"))
		}
	} else {
		// prior_notice_last_day is forbidden for booking_type=0 and 1
		if ent.PriorNoticeLastDay.Valid {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("prior_notice_last_day", ent.PriorNoticeLastDay.String(), "prior_notice_last_day is forbidden for booking_type!=2"))
		}
	}

	// booking_type=0: prior_notice_duration_max is forbidden
	if bookingType == 0 {
		if ent.PriorNoticeDurationMax.Valid {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("prior_notice_duration_max", ent.PriorNoticeDurationMax.String(), "prior_notice_duration_max is forbidden for booking_type=0"))
		}
		// prior_notice_start_day is forbidden for booking_type=0
		if ent.PriorNoticeStartDay.Valid {
			errs = append(errs, causes.NewConditionallyForbiddenFieldError("prior_notice_start_day", ent.PriorNoticeStartDay.String(), "prior_notice_start_day is forbidden for booking_type=0"))
		}
	}

	// prior_notice_last_time requires prior_notice_last_day
	if ent.PriorNoticeLastTime.Valid && !ent.PriorNoticeLastDay.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("prior_notice_last_day"))
	}
	// prior_notice_last_day requires prior_notice_last_time
	if ent.PriorNoticeLastDay.Valid && !ent.PriorNoticeLastTime.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("prior_notice_last_time"))
	}

	// prior_notice_start_time requires prior_notice_start_day
	if ent.PriorNoticeStartTime.Valid && !ent.PriorNoticeStartDay.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("prior_notice_start_day"))
	}
	// prior_notice_start_day requires prior_notice_start_time
	if ent.PriorNoticeStartDay.Valid && !ent.PriorNoticeStartTime.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("prior_notice_start_time"))
	}

	// prior_notice_service_id is forbidden except for booking_type=2
	if bookingType != 2 && ent.PriorNoticeServiceID.Valid {
		errs = append(errs, causes.NewConditionallyForbiddenFieldError("prior_notice_service_id", ent.PriorNoticeServiceID.Val, "prior_notice_service_id is only allowed for booking_type=2"))
	}

	// booking_type=1: prior_notice_start_day forbidden if prior_notice_duration_max is defined
	if bookingType == 1 && ent.PriorNoticeDurationMax.Valid && ent.PriorNoticeStartDay.Valid {
		errs = append(errs, causes.NewConditionallyForbiddenFieldError("prior_notice_start_day", ent.PriorNoticeStartDay.String(), "prior_notice_start_day is forbidden when prior_notice_duration_max is defined for booking_type=1"))
	}

	return errs
}
