-- route_attributes.txt
insert into ext_plus_route_attributes(route_id,feed_version_id,category,subcategory,running_way) values (
    (select r.id from gtfs_routes r join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and route_id = '01'),
    (select r.feed_version_id from gtfs_routes r join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and route_id = '01'),
    2,
    201,
    1
);

-- working stop ref
insert into tl_stop_external_references(stop_id,feed_version_id,target_feed_onestop_id,target_stop_id) values (
    (select s.id as stop_id from gtfs_stops s join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and stop_id = 'FTVL'),
    (select s.feed_version_id from gtfs_stops s join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and stop_id = 'FTVL'),    
    'CT',
    '70041'
);

-- broken stop ref
insert into tl_stop_external_references(stop_id,feed_version_id,target_feed_onestop_id,target_stop_id) values (
    (select s.id as stop_id from gtfs_stops s join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and stop_id = 'POWL'),
    (select s.feed_version_id from gtfs_stops s join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and stop_id = 'POWL'),
    'CT',
    'missing'
);

-- stop obs 1
insert into ext_performance_stop_observations(id,feed_version_id,source,trip_start_date,from_stop_id,to_stop_id,trip_id,route_id,observed_arrival_time,observed_departure_time) values (
    (select s.id from gtfs_stops s join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and stop_id = 'FTVL'),
    (select s.feed_version_id from gtfs_stops s join feed_states fs using(feed_version_id) join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'BA' and stop_id = 'FTVL'),
    'TripUpdate',
    '2023-03-09'::date,
    'LAKE',
    'FTVL',
    'test',
    '03',
    36000,
    36010
);

--- route segments
insert into tl_segments(id,feed_version_id,way_id,geometry) values 
    (
        1418704,
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'),
        645693994,
        '0102000020E610000002000000956247E3509D54C0CC423BA759F43B404208C897509D54C0C8ED974F56F43B40'
    ),
    (
        1418711,
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'),
        90865590,
        '0102000020E61000000E0000007B6B60AB049C54C06420CF2EDF0E3C404ED4D2DC0A9C54C0A60F5D50DF0E3C40185FB4C70B9C54C02331410DDF0E3C40B265F9BA0C9C54C09A95ED43DE0E3C40C3B645990D9C54C04D2CF015DD0E3C40931804560E9C54C09CFD8172DB0E3C40DBC4C9FD0E9C54C0431A1538D90E3C40B1FD648C0F9C54C04582A966D60E3C402D05A4FD0F9C54C023145B41D30E3C4098A1F144109C54C01FBFB7E9CF0E3C40DA907F66109C54C03883BF5FCC0E3C4092CA1473109C54C0342E1C08C90E3C4092CA1473109C54C0D68D7747C60E3C4092CA1473109C54C018601F9DBA0E3C40'
    );
select setval('tl_segments_id_seq', (select max(id)+1 from tl_segments), false);

insert into tl_segment_patterns(feed_version_id,segment_id,route_id,shape_id,stop_pattern_id,way_id,direction_id,sequence_idx) values
    (
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'), -- EG
        1418704,
        (select id from gtfs_routes where route_long_name = '22nd Street'),
        (select id from gtfs_shapes where shape_id = '41698'),
        42,
        645693994,
        0,
        0
    ),
    (
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'), -- EG
        1418711,
        (select id from gtfs_routes where route_long_name = '22nd Street'),
        (select id from gtfs_shapes where shape_id = '41698'),
        42,
        90865590,
        0,
        1
    ),    
    (
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'), -- EG
        1418704,
        (select id from gtfs_routes where route_long_name = 'South Tampa'),
        (select id from gtfs_shapes where shape_id = '41708'),
        40,
        645693994,
        0,
        2
    );    

-- GTFS Fares v2 for CT. The test feeds have no networks, areas, timeframes, or
-- fare_leg_join_rules, so add a small synthetic set to the active CT feed version.
-- Text reference columns hold GTFS ids, as the importer writes them.
create temporary table ct_fv as
    select fs.feed_version_id as id from feed_states fs join current_feeds cf on cf.id = fs.feed_id where cf.onestop_id = 'CT';

insert into gtfs_networks(feed_version_id, network_id, network_name)
    select id, 'local', 'Local service' from ct_fv
    union all select id, 'express', 'Express service' from ct_fv;

insert into gtfs_route_networks(feed_version_id, network_id, route_id)
    select r.feed_version_id, n.id, r.id
    from gtfs_routes r
    join ct_fv on ct_fv.id = r.feed_version_id
    join gtfs_networks n on n.feed_version_id = r.feed_version_id and n.network_id = (case when r.route_id = 'Bu-130' then 'express' else 'local' end)
    where r.route_id in ('Lo-130', 'Bu-130');

insert into gtfs_areas(feed_version_id, area_id, area_name)
    select id, 'zone1', 'Zone 1' from ct_fv
    union all select id, 'zone4', 'Zone 4' from ct_fv;

insert into gtfs_stop_areas(feed_version_id, area_id, stop_id)
    select s.feed_version_id, a.id, s.id
    from gtfs_stops s
    join ct_fv on ct_fv.id = s.feed_version_id
    join gtfs_areas a on a.feed_version_id = s.feed_version_id and a.area_id = (case when s.stop_id in ('70011', '70012') then 'zone1' else 'zone4' end)
    where s.stop_id in ('70011', '70012', '70261', '70262');

insert into gtfs_timeframes(feed_version_id, timeframe_group_id, start_time, end_time, service_id)
    select c.feed_version_id, 'weekday_peak', 21600, 32400, c.id
    from gtfs_calendars c join ct_fv on ct_fv.id = c.feed_version_id where c.service_id = 'mtwtf';

insert into gtfs_rider_categories(feed_version_id, rider_category_id, rider_category_name, is_default_fare_category, eligibility_url, min_age, max_age)
    select id, 'adult', 'Adult', 1, null, null, null from ct_fv
    union all select id, 'youth', 'Youth', 0, 'https://www.caltrain.com/fares', 5, 18 from ct_fv;

insert into gtfs_fare_products(feed_version_id, fare_product_id, fare_product_name, amount, currency, rider_category_id, duration_start, duration_amount, duration_unit, duration_type)
    select id, 'two_zone', 'Two zones', 6.40, 'USD', 'adult', null::int, null::real, null::int, null::int from ct_fv
    union all select id, 'two_zone', 'Two zones', 3.20, 'USD', 'youth', null, null, null, null from ct_fv
    union all select id, 'two_zone_peak', 'Two zones, peak', 7.40, 'USD', 'adult', null, null, null, null from ct_fv
    union all select id, 'express_upgrade', 'Express upgrade', 1.00, 'USD', null, null, null, null, null from ct_fv
    union all select id, 'day_pass', 'Day pass', 15.00, 'USD', 'adult', 0, 1, 3, 1 from ct_fv;

insert into gtfs_fare_leg_rules(feed_version_id, leg_group_id, network_id, from_area_id, to_area_id, from_timeframe_group_id, to_timeframe_group_id, fare_product_id, rule_priority, transfer_only)
    select id, 'ct_local', 'local', 'zone1', 'zone4', null, null, 'two_zone', 0, null::int from ct_fv
    union all select id, 'ct_local', 'local', 'zone1', 'zone4', 'weekday_peak', null, 'two_zone_peak', 1, null::int from ct_fv
    union all select id, 'ct_express', 'express', null, null, null, null, 'two_zone', 0, 1 from ct_fv;

insert into gtfs_fare_transfer_rules(feed_version_id, from_leg_group_id, to_leg_group_id, transfer_count, duration_limit, duration_limit_type, fare_transfer_type, fare_product_id, filter_fare_product_id)
    select id, 'ct_local', 'ct_express', 1, 5400, 1, 0, 'express_upgrade', 'two_zone' from ct_fv;

insert into gtfs_fare_leg_join_rules(feed_version_id, from_network_id, to_network_id, from_stop_id, to_stop_id)
    select id, 'local', 'express', '70261', '70262' from ct_fv;

drop table ct_fv;

-- unactivate feed
update feed_states set feed_version_id = null, materialized_feed_version_id = null, active_feed_version_id = null where feed_id = (select id from current_feeds where onestop_id = 'EX');

-- set public
update feed_states set public = false;
update feed_states set public = true where id in (select id from current_feeds where onestop_id != 'EG');

insert into tl_tenants(tenant_name) values ('tl-tenant');
insert into tl_tenants(tenant_name) values ('restricted-tenant');
insert into tl_tenants(tenant_name) values ('all-users-tenant');

insert into tl_groups(group_name) values ('CT-group');
insert into tl_groups(group_name) values ('BA-group');
insert into tl_groups(group_name) values ('HA-group');
insert into tl_groups(group_name) values ('EX-group');
insert into tl_groups(group_name) values ('test-group');

