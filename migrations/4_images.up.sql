-- Table: public.image

-- DROP TABLE public.image;

CREATE TABLE public.image
(
    id uuid NOT NULL,
    related_id text COLLATE pg_catalog."default",
    image_type integer,
    image_src text COLLATE pg_catalog."default" NOT NULL,
    is_deleted boolean,
    insert_time timestamp without time zone NOT NULL,
    update_time timestamp without time zone NULL,
    user_id integer NOT NULL,
    CONSTRAINT user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public."user" (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT image_pkey PRIMARY KEY (id),
    CONSTRAINT image_src_uniquekey UNIQUE (image_src)
)