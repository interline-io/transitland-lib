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

Realtime data naming particular runs of trips at Fruitvale ("FTVL"), each delay-only at that stop:

- Trip "1031527WKDY" has trip updates dated 2018-05-30 (60 second delay), 2018-05-31 (300 seconds) and 2030-05-28 (90 seconds, a date answered from the fallback week).
- Trips "2211533WKDY" (120 seconds) and "5172328WKDY" (180 seconds, departing 24:02 on its service date) have undated trip updates, which describe their current runs.
- Trip "1031527WKDY" has alerts dated 2018-05-30, 2018-05-31 and 2030-05-28, and undated alerts in force during its 2018-05-30 run, during all of 2018-06-05, and always.
- Trip "5172328WKDY" has an alert dated 2018-05-30, and an undated alert in force from midnight to 1 am on 2018-05-31, during its run of 2018-05-30.
- Vehicles run trip "1031527WKDY" dated 2018-05-30, and trip "2211533WKDY" undated.
