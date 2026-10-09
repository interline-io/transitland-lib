# BA.json

Synthetic RT data for a selection of trips from the BA test feed on 2018-05-30, with a delay of 30 seconds

# BA-added.json

Trip "1011630WKDY" is normal but with 31 second delay
Trip "-123" is added (based on trip "1031527WKDY" with 32 second delay)
Trip "1031645WKDY" is canceled

# CT.json

Synthetic RT data for a selection of trips from the CT test feed on 2018-05-30, with a delay of 30 seconds

# BA-alerts-informed-entity.json

Alerts on route "05" at stop "FTVL", on route "05" at stop "12TH", on trip "1031527WKDY" with every entity selector field set, on BART's subway routes by route_type, and on every bus route by route_type alone

# BA-runs-trip-updates.json, BA-runs-alerts.json, BA-runs-vehicle-positions.json

Realtime data naming particular runs of trips at Fruitvale ("FTVL"), each delay-only at that stop. A message with a start_date describes the run on that date; one without describes the trip's current run.

- Trip "1031527WKDY" has trip updates dated 2018-05-30 (60 second delay), 2018-05-31 (300 seconds) and 2030-05-28 (90 seconds, a date answered from the fallback week).
- Trips "2211533WKDY" (120 seconds), "5172328WKDY" (180 seconds, departing FTVL at 24:02 and arriving at its last stop at 24:31) and "2290403WKDY" (240 seconds, starting at 04:03 and departing FTVL at 04:32) have undated trip updates.
- Trip "ADDED1", added in real time on route "05", has trip updates dated 2018-05-31 and 2018-06-01, each departing FTVL at 16:03.
- Trip "1031527WKDY" has alerts dated 2018-05-30, 2018-05-31 and 2030-05-28, and an undated alert.
- Trip "5172328WKDY" has an alert dated 2018-05-30 and an undated alert.
- Vehicles run trip "1031527WKDY" dated 2018-05-30, and trip "2211533WKDY" undated.
