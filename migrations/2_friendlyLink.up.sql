-- Table: public.friendly_link

-- DROP TABLE public.friendly_link;

CREATE TABLE public.friendly_link
(
    id uuid NOT NULL,
    description character varying COLLATE pg_catalog."default" NOT NULL,
    website_url character varying COLLATE pg_catalog."default" NOT NULL,
    friendly_link_page_url character varying COLLATE pg_catalog."default" NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    access_time timestamp without time zone,
    is_approved boolean NOT NULL,
    is_deleted boolean NOT NULL,
    website_name character varying COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT friendly_link_pkey PRIMARY KEY (id)
)