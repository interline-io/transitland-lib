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

insert into tl_segment_patterns(feed_version_id,segment_id,route_id,shape_id,stop_pattern_id,way_id,direction_id) values
    (
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'), -- EG
        1418704,
        (select id from gtfs_routes where route_long_name = '22nd Street'),
        (select id from gtfs_shapes where shape_id = '41698'),
        42,
        645693994,
        0
    ),
    (
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'), -- EG
        1418711,
        (select id from gtfs_routes where route_long_name = '22nd Street'),
        (select id from gtfs_shapes where shape_id = '41698'),
        42,
        645693994,
        0
    ),    
    (
        (select feed_version_id from gtfs_agencies where agency_name = 'Hillsborough Area Regional Transit'), -- EG
        1418704,
        (select id from gtfs_routes where route_long_name = 'South Tampa'),
        (select id from gtfs_shapes where shape_id = '41708'),
        40,
        645693994,
        0
    );    

-- unactivate feed
update feed_states set feed_version_id = null where feed_id = (select id from current_feeds where onestop_id = 'EX');

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

