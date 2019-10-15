package dmfr

// func hstoreToMap(h postgres.Hstore) map[string]string {
// 	m := map[string]string{}
// 	for k, v := range h {
// 		m[k] = *v
// 	}
// 	return m
// }

// func mapToHstore(m map[string]string) postgres.Hstore {
// 	h := postgres.Hstore{}
// 	for k, v := range m {
// 		b := v
// 		h[k] = &b
// 	}
// 	return h
// }

// func pqToString(a pq.StringArray) []string {
// 	ret := []string{}
// 	for _, i := range a {
// 		ret = append(ret, i)
// 	}
// 	return ret
// }

// func stringToPq(a []string) pq.StringArray {
// 	ret := pq.StringArray{}
// 	for _, i := range a {
// 		ret = append(ret, i)
// 	}
// 	return ret
// }

// // dbFeed hides database specific things...
// type dbFeed struct {
// 	URLs            postgres.Hstore
// 	Authorization   postgres.Hstore
// 	License         postgres.Hstore
// 	OtherIDs        postgres.Hstore
// 	AssociatedFeeds pq.StringArray
// 	Languages       pq.StringArray
// 	Feed
// }

// func (m *dbFeed) To() *Feed {
// 	f := m.Feed
// 	f.URLs = hstoreToMap(m.URLs)
// 	f.Authorization = hstoreToMap(m.Authorization)
// 	f.License = hstoreToMap(m.License)
// 	f.OtherIDs = hstoreToMap(m.OtherIDs)
// 	f.Languages = pqToString(m.Languages)
// 	f.AssociatedFeeds = pqToString(m.AssociatedFeeds)
// 	return &f
// }

// func (m *dbFeed) From(f *Feed) {
// 	m.Feed = *f
// 	if f == nil {
// 		return
// 	}
// 	m.URLs = mapToHstore((f.URLs))
// 	m.Authorization = mapToHstore(f.Authorization)
// 	m.License = mapToHstore(f.License)
// 	m.OtherIDs = mapToHstore(f.OtherIDs)
// 	m.Languages = stringToPq(f.Languages)
// 	m.AssociatedFeeds = stringToPq(f.AssociatedFeeds)
// }

// MainSync .
// func MainSync(tx *gorm.DB, filenames []string) ([]string, error) {
// 	found := []string{}
// 	// Load
// 	regs := []*dmfr.Registry{}
// 	for _, fn := range filenames {
// 		reg, err := dmfr.LoadAndParseRegistry(fn)
// 		if err != nil {
// 			return found, err
// 		}
// 		regs = append(regs, reg)
// 	}
// 	// Import
// 	for _, registry := range regs {
// 		fids, err := ImportRegistry(tx, registry)
// 		if err != nil {
// 			return found, err
// 		}
// 		found = append(found, fids...)
// 	}
// 	// Hide
// 	if err := HideUnusedFeeds(tx, found); err != nil {
// 		return found, err
// 	}
// 	return found, nil
// }

// // ImportRegistry .
// func ImportRegistry(db *gorm.DB, registry *dmfr.Registry) ([]string, error) {
// 	// Update feeds from DMFR
// 	feedids := []string{}
// 	for _, rfeed := range registry.Feeds {
// 		// Create a new Feed from the Registry Feed
// 		fmt.Println("registry feed:", rfeed.ID)
// 		tf := NewFeedFromDMFR(&rfeed)
// 		// Check if we have the existing Feed
// 		df := dbFeed{}
// 		if err := db.Where("onestop_id = ?", tf.OnestopID).Find(&df).Error; err == nil {
// 			// Explicitly preserve these values, and only these values
// 			fmt.Println("updating existing feed:", df.ID)
// 			tf.LastFetchedAt = df.LastFetchedAt
// 			tf.LastImportedAt = df.LastImportedAt
// 			tf.LastSuccessfulFetchAt = df.LastSuccessfulFetchAt
// 			tf.ActiveFeedVersionID = df.ActiveFeedVersionID
// 			tf.LastFetchError = df.LastFetchError
// 			tf.ID = df.ID
// 		} else {
// 			fmt.Println("new feed")
// 		}
// 		df.From(tf) // convert to postgres happy types
// 		// Save back to database
// 		if err := db.Save(&df).Error; err != nil {
// 			return []string{}, err
// 		}
// 		feedids = append(feedids, rfeed.ID)
// 	}
// 	return feedids, nil
// }

// // HideUnusedFeeds .
// func HideUnusedFeeds(db *gorm.DB, found []string) error {
// 	// Delete unreferenced feeds
// 	if err := db.Model(&Feed{}).Where("onestop_id NOT IN (?)", found).Update("DeletedAt", time.Now()).Error; err != nil {
// 		return err
// 	}
// 	return nil
// }
