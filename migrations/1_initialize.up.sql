-- Table: public."user"

-- DROP TABLE public."user";

CREATE TABLE public."user"
(
    id serial NOT NULL,
    user_name character varying(20) COLLATE pg_catalog."default" NOT NULL,
    level2pwd character varying(200) COLLATE pg_catalog."default",
    insert_time timestamp without time zone NOT NULL,
    update_time timestamp without time zone,
    is_banned boolean NOT NULL,
    op_issuer character varying(100) COLLATE pg_catalog."default" NOT NULL,
    op_userid character varying(100) COLLATE pg_catalog."default" NOT NULL,
    avatar character varying(100) COLLATE pg_catalog."default" NOT NULL,
    email character varying(50) COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT user_pkey PRIMARY KEY (id),
    CONSTRAINT email_uniquekey UNIQUE (email)
,
    CONSTRAINT user_name_uniquekey UNIQUE (user_name)

);

-- Table: public.category

-- DROP TABLE public.category;

CREATE TABLE public.category
(
    id serial NOT NULL,
    name character varying(20) COLLATE pg_catalog."default" NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    update_time timestamp without time zone,
    is_deleted boolean NOT NULL,
    user_id integer NOT NULL,
    CONSTRAINT category_pkey PRIMARY KEY (id),
    CONSTRAINT user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public."user" (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
);

-- Table: public.article

-- DROP TABLE public.article;

CREATE TABLE public.article
(
    id serial NOT NULL,
    category_id integer NOT NULL,
    content text COLLATE pg_catalog."default" NOT NULL,
    html text COLLATE pg_catalog."default" NOT NULL,
    insert_time timestamp without time zone NOT NULL,
    is_banned boolean NOT NULL,
    is_deleted boolean NOT NULL,
    title text COLLATE pg_catalog."default" NOT NULL,
    type integer NOT NULL,
    update_time timestamp without time zone,
    user_id integer NOT NULL,
    cover character varying(100) COLLATE pg_catalog."default",
    summary text COLLATE pg_catalog."default" NOT NULL,
    backup_article_id integer,
    remark character varying(20) COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT article_pkey PRIMARY KEY (id),
    CONSTRAINT backup_article_id_fkey FOREIGN KEY (backup_article_id)
        REFERENCES public.article (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID,
    CONSTRAINT category_id_fkey FOREIGN KEY (category_id)
        REFERENCES public.category (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID,
    CONSTRAINT user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public."user" (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
);