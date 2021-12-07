package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestOnestopIDBuilder(t *testing.T) {
	type hw = map[string][]string
	type testcase struct {
		Name string
		Hits hw
	}
	type testgroup struct {
		URL   string
		Cases []testcase
	}
	groups := map[string]testgroup{
		"ExampleFeed": {
			testutil.ExampleZip.URL,
			[]testcase{
				{"o-9qs-demotransitauthority", hw{"o-9qs-demotransitauthority": []string{"DTA"}}},
				{"r-9qsb-20", hw{"r-9qsb-20": []string{"BFC"}}},
				{"r-9qscy-10", hw{"r-9qscy-10": []string{"AB"}}},
				{"r-9qscy-30", hw{"r-9qscy-30": []string{"STBA"}}},
				{"r-9qsczp-40", hw{"r-9qsczp-40": []string{"CITY"}}},
				{"r-9qt1-50", hw{"r-9qt1-50": []string{"AAMV"}}},
				{"s-9qkxnx40xt-furnacecreekresortdemo", hw{"s-9qkxnx40xt-furnacecreekresortdemo": []string{"FUR_CREEK_RES"}}},
				{"s-9qscv9zzb5-bullfrogdemo", hw{"s-9qscv9zzb5-bullfrogdemo": []string{"BULLFROG"}}},
				{"s-9qscwx8n60-nyecountyairportdemo", hw{"s-9qscwx8n60-nyecountyairportdemo": []string{"BEATTY_AIRPORT"}}},
				{"s-9qscyz5vqg-doingave~davendemo", hw{"s-9qscyz5vqg-doingave~davendemo": []string{"DADAN"}}},
				{"s-9qsczn2rk0-emainst~sirvingstdemo", hw{"s-9qsczn2rk0-emainst~sirvingstdemo": []string{"EMSI"}}},
				{"s-9qsfnb5uz6-northave~davendemo", hw{"s-9qsfnb5uz6-northave~davendemo": []string{"NADAV"}}},
				{"s-9qsfp00vhs-northave~naavedemo", hw{"s-9qsfp00vhs-northave~naavedemo": []string{"NANAA"}}},
				{"s-9qsfp2212t-stagecoachhotel~casinodemo", hw{"s-9qsfp2212t-stagecoachhotel~casinodemo": []string{"STAGECOACH"}}},
				{"s-9qt0rnrkjt-amargosavalleydemo", hw{"s-9qt0rnrkjt-amargosavalleydemo": []string{"AMV"}}},
			},
		},
		"Caltrain": {
			testutil.ExampleFeedCaltrain.URL,
			[]testcase{
				{"o-9q9-caltrain", hw{"o-9q9-caltrain": []string{"caltrain-ca-us"}}},
				{"r-9q9-limited", hw{"r-9q9-limited": []string{"Li-130"}}},
				{"r-9q9-local", hw{"r-9q9-local": []string{"Lo-130"}}},
				{"r-9q9j-bullet", hw{"r-9q9j-bullet": []string{"Bu-130"}}},
				{"r-9q9j-giantsspecial", hw{"r-9q9j-giantsspecial": []string{"Gi-130"}}},
				{"r-9q9j-special", hw{"r-9q9j-special": []string{"Sp-130"}}},
				{"r-9q9k6-tasj~shuttle", hw{"r-9q9k6-tasj~shuttle": []string{"TaSj-130"}}},
				{"s-9q8vzcr03n-burlingamecaltrain", hw{"s-9q8vzcr03n-burlingamecaltrain": []string{"70082"}}},
				{"s-9q8vzcr0t0-burlingamecaltrain", hw{"s-9q8vzcr0t0-burlingamecaltrain": []string{"70081"}}},
				{"s-9q8vzdd5q5-broadwaycaltrain", hw{"s-9q8vzdd5q5-broadwaycaltrain": []string{"70072"}}},
				{"s-9q8vzdd5ze-broadwaycaltrain", hw{"s-9q8vzdd5ze-broadwaycaltrain": []string{"70071"}}},
				{"s-9q8vzhbdsn-millbraecaltrain", hw{"s-9q8vzhbdsn-millbraecaltrain": []string{"70062"}}},
				{"s-9q8vzhbg0m-millbraecaltrain", hw{"s-9q8vzhbg0m-millbraecaltrain": []string{"70061"}}},
			},
		},
		"BART": {
			testutil.ExampleFeedBART.URL,
			[]testcase{
				{"o-9q9-bayarearapidtransit", hw{"o-9q9-bayarearapidtransit": []string{"BART"}}},
				{"r-9q8y-richmond~dalycity~millbrae", hw{"r-9q8y-richmond~dalycity~millbrae": []string{"07"}}},
				{"r-9q9-antioch~sfia~millbrae", hw{"r-9q9-antioch~sfia~millbrae": []string{"01"}}},
				{"r-9q9n-dublin~pleasanton~dalycity", hw{"r-9q9n-dublin~pleasanton~dalycity": []string{"11"}}},
				{"r-9q9n-warmsprings~southfremont~dalycity", hw{"r-9q9n-warmsprings~southfremont~dalycity": []string{"05"}}},
				{"r-9q9n-warmsprings~southfremont~richmond", hw{"r-9q9n-warmsprings~southfremont~richmond": []string{"03"}}},
				{"r-9q9ne-oaklandairport~coliseum", hw{"r-9q9ne-oaklandairport~coliseum": []string{"19"}}},
				{"s-9q8vyzu8fh-sanfranciscointernationalairport", hw{"s-9q8vyzu8fh-sanfranciscointernationalairport": []string{"SFIA"}}},
				{"s-9q8vzhbtrn-millbrae", hw{"s-9q8vzhbtrn-millbrae": []string{"MLBR"}}},
				{"s-9q9p1dy9x5-19thstoakland", hw{"s-9q9p1dy9x5-19thstoakland": []string{"19TH_N", "19TH"}}},
				{"s-9q9p1wxf72-macarthur", hw{"s-9q9p1wxf72-macarthur": []string{"MCAR_S", "MCAR"}}},
				{"s-9q8ym8x40u-southsanfrancisco", hw{"s-9q8ym8x40u-southsanfrancisco": []string{"SSAN"}}},
				{"s-9q8ymhqbcv-colma", hw{"s-9q8ymhqbcv-colma": []string{"COLM"}}},
				{"s-9q8yteh2nm-balboapark", hw{"s-9q8yteh2nm-balboapark": []string{"BALB"}}},
				{"s-9q8ytvn7j2-glenpark", hw{"s-9q8ytvn7j2-glenpark": []string{"GLEN"}}},
				{"s-9q8yy29u4d-24thstmission", hw{"s-9q8yy29u4d-24thstmission": []string{"24TH"}}},
				{"s-9q8yy6btqm-16thstmission", hw{"s-9q8yy6btqm-16thstmission": []string{"16TH"}}},
				{"s-9q8yymsfbh-civiccenter~unplaza", hw{"s-9q8yymsfbh-civiccenter~unplaza": []string{"CIVC"}}},
			},
		},
	}
	for groupName, testGroup := range groups {
		t.Run(groupName, func(t *testing.T) {
			cp, writer, err := newMockCopier(testGroup.URL)
			if err != nil {
				t.Fatal(err)
			}
			e := NewOnestopIDBuilder()
			cp.AddExtension(e)
			cpr := cp.Copy()
			if cpr.WriteError != nil {
				t.Fatal(err)
			}
			hits := hw{}
			for _, ent := range writer.Reader.OtherList {
				switch v := ent.(type) {
				case *StopOnestopID:
					hits[v.OnestopID] = append(hits[v.OnestopID], v.StopID)
				case *RouteOnestopID:
					hits[v.OnestopID] = append(hits[v.RouteID], v.RouteID)
				case *AgencyOnestopID:
					hits[v.OnestopID] = append(hits[v.AgencyID], v.AgencyID)
				}
			}
			// for k, v := range hits {
			// 	fmt.Printf(`{"%s", hw{"%s":%#v}},`+"\n", k, k, v)
			// }
			for _, tc := range testGroup.Cases {
				t.Run(tc.Name, func(t *testing.T) {
					for k, v := range tc.Hits {
						assert.ElementsMatch(t, hits[k], v)
					}
				})
			}

		})
	}
}
