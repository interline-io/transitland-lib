CREATE FUNCTION public.tl_onestop_regex(v text) RETURNS text
    LANGUAGE plpgsql STABLE
    AS $$
BEGIN
RETURN regexp_replace(regexp_replace(lower(v), '[\-\:\&\@\/]', '~', 'g'), '[^[:alnum:]\~\>\<]', '', 'g');
END;
$$;
