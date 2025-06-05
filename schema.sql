--
-- PostgreSQL database dump
--

-- Dumped from database version 17.4 (Debian 17.4-1.pgdg120+2)
-- Dumped by pg_dump version 17.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: users; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA users;


ALTER SCHEMA users OWNER TO postgres;

--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;


--
-- Name: EXTENSION vector; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION vector IS 'vector data type and ivfflat and hnsw access methods';


--
-- Name: party; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.party AS ENUM (
    'none',
    'KMT',
    'DPP',
    'TPP'
);


ALTER TYPE public.party OWNER TO postgres;

--
-- Name: source_type; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.source_type AS ENUM (
    'url',
    'text'
);


ALTER TYPE public.source_type OWNER TO postgres;

--
-- Name: task_status; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.task_status AS ENUM (
    'pending',
    'processing',
    'done',
    'failed'
);


ALTER TYPE public.task_status OWNER TO postgres;

--
-- Name: avg_embedding(integer, integer); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.avg_embedding(aid integer, mid integer) RETURNS public.vector
    LANGUAGE sql
    AS $_$
SELECT CASE
        WHEN COUNT(*) = 0 THEN NULL
        ELSE AVG(vector)
    END
FROM embeddings
WHERE article_id = $1
    AND model_id = $2
    AND vector IS NOT NULL;
$_$;


ALTER FUNCTION public.avg_embedding(aid integer, mid integer) OWNER TO postgres;

--
-- Name: concat_article_chunks(integer); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.concat_article_chunks(aid integer) RETURNS text
    LANGUAGE sql
    AS $_$
SELECT STRING_AGG(
        substring(
            c.content
            FROM c.start_at + 1 FOR c.end_at - c.start_at
        ),
        '\n'
        ORDER BY c.ord
    )
FROM chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0;
$_$;


ALTER FUNCTION public.concat_article_chunks(aid integer) OWNER TO postgres;

--
-- Name: none_overlap_chunk(integer); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.none_overlap_chunk(aid integer) RETURNS TABLE(chunk_id integer, ord integer, content text)
    LANGUAGE sql
    AS $_$
SELECT c.id AS chunk_id,
    c.ord AS ord,
    substring(
        c.content
        FROM c.start_at + 1 FOR c.end_at - c.start_at
    ) AS content
FROM chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0
ORDER BY c.ord;
$_$;


ALTER FUNCTION public.none_overlap_chunk(aid integer) OWNER TO postgres;

--
-- Name: avg_embedding(integer, integer); Type: FUNCTION; Schema: users; Owner: postgres
--

CREATE FUNCTION users.avg_embedding(aid integer, mid integer) RETURNS public.vector
    LANGUAGE sql
    AS $_$
SELECT CASE
        WHEN COUNT(*) = 0 THEN NULL
        ELSE AVG(vector)
    END
FROM users.embeddings
WHERE article_id = $1
    AND model_id = $2
    AND vector IS NOT NULL;
$_$;


ALTER FUNCTION users.avg_embedding(aid integer, mid integer) OWNER TO postgres;

--
-- Name: concat_article_chunks(integer); Type: FUNCTION; Schema: users; Owner: postgres
--

CREATE FUNCTION users.concat_article_chunks(aid integer) RETURNS text
    LANGUAGE sql
    AS $_$
SELECT STRING_AGG(
        substring(
            c.content
            FROM c.start_at + 1 FOR c.end_at - c.start_at
        ),
        '\n'
        ORDER BY c.ord
    )
FROM users.chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0;
$_$;


ALTER FUNCTION users.concat_article_chunks(aid integer) OWNER TO postgres;

--
-- Name: none_overlap_chunk(integer); Type: FUNCTION; Schema: users; Owner: postgres
--

CREATE FUNCTION users.none_overlap_chunk(aid integer) RETURNS TABLE(chunk_id integer, ord integer, content text)
    LANGUAGE sql
    AS $_$
SELECT c.id AS chunk_id,
    c.ord AS ord,
    substring(
        c.content
        FROM c.start_at + 1 FOR c.end_at - c.start_at
    ) AS content
FROM users.chunks c
WHERE c.article_id = $1
    AND character_length(c.content) > 0
ORDER BY c.ord;
$_$;


ALTER FUNCTION users.none_overlap_chunk(aid integer) OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: articles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.articles (
    id integer NOT NULL,
    title text NOT NULL,
    url text NOT NULL,
    source text NOT NULL,
    md5 text NOT NULL,
    party public.party DEFAULT 'none'::public.party NOT NULL,
    published_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.articles OWNER TO postgres;

--
-- Name: articles_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.articles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.articles_id_seq OWNER TO postgres;

--
-- Name: articles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.articles_id_seq OWNED BY public.articles.id;


--
-- Name: articles_keywords; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.articles_keywords (
    article_id integer NOT NULL,
    keyword_id integer NOT NULL
);


ALTER TABLE public.articles_keywords OWNER TO postgres;

--
-- Name: chunks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chunks (
    id integer NOT NULL,
    article_id integer NOT NULL,
    content text NOT NULL,
    ord integer NOT NULL,
    start_at integer NOT NULL,
    end_at integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.chunks OWNER TO postgres;

--
-- Name: chunks_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.chunks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.chunks_id_seq OWNER TO postgres;

--
-- Name: chunks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.chunks_id_seq OWNED BY public.chunks.id;


--
-- Name: embeddings; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.embeddings (
    id integer NOT NULL,
    article_id integer NOT NULL,
    chunk_id integer NOT NULL,
    model_id integer NOT NULL,
    vector public.vector(1024) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.embeddings OWNER TO postgres;

--
-- Name: embeddings_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.embeddings_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.embeddings_id_seq OWNER TO postgres;

--
-- Name: embeddings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.embeddings_id_seq OWNED BY public.embeddings.id;


--
-- Name: keywords; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.keywords (
    id integer NOT NULL,
    term character varying(32) NOT NULL
);


ALTER TABLE public.keywords OWNER TO postgres;

--
-- Name: keywords_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.keywords_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.keywords_id_seq OWNER TO postgres;

--
-- Name: keywords_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.keywords_id_seq OWNED BY public.keywords.id;


--
-- Name: models; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.models (
    id integer NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.models OWNER TO postgres;

--
-- Name: models_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.models_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.models_id_seq OWNER TO postgres;

--
-- Name: models_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.models_id_seq OWNED BY public.models.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- Name: articles; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.articles (
    id integer NOT NULL,
    task_id uuid NOT NULL,
    title text NOT NULL,
    url text NOT NULL,
    source text NOT NULL,
    md5 text NOT NULL,
    party public.party DEFAULT 'none'::public.party NOT NULL,
    published_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE users.articles OWNER TO postgres;

--
-- Name: articles_id_seq; Type: SEQUENCE; Schema: users; Owner: postgres
--

CREATE SEQUENCE users.articles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE users.articles_id_seq OWNER TO postgres;

--
-- Name: articles_id_seq; Type: SEQUENCE OWNED BY; Schema: users; Owner: postgres
--

ALTER SEQUENCE users.articles_id_seq OWNED BY users.articles.id;


--
-- Name: articles_keywords; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.articles_keywords (
    article_id integer NOT NULL,
    keyword_id integer NOT NULL
);


ALTER TABLE users.articles_keywords OWNER TO postgres;

--
-- Name: chunks; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.chunks (
    id integer NOT NULL,
    article_id integer NOT NULL,
    content text NOT NULL,
    ord integer NOT NULL,
    start_at integer NOT NULL,
    end_at integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE users.chunks OWNER TO postgres;

--
-- Name: chunks_id_seq; Type: SEQUENCE; Schema: users; Owner: postgres
--

CREATE SEQUENCE users.chunks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE users.chunks_id_seq OWNER TO postgres;

--
-- Name: chunks_id_seq; Type: SEQUENCE OWNED BY; Schema: users; Owner: postgres
--

ALTER SEQUENCE users.chunks_id_seq OWNED BY users.chunks.id;


--
-- Name: embeddings; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.embeddings (
    id integer NOT NULL,
    article_id integer NOT NULL,
    chunk_id integer NOT NULL,
    model_id integer NOT NULL,
    vector public.vector(1024) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE users.embeddings OWNER TO postgres;

--
-- Name: embeddings_id_seq; Type: SEQUENCE; Schema: users; Owner: postgres
--

CREATE SEQUENCE users.embeddings_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE users.embeddings_id_seq OWNER TO postgres;

--
-- Name: embeddings_id_seq; Type: SEQUENCE OWNED BY; Schema: users; Owner: postgres
--

ALTER SEQUENCE users.embeddings_id_seq OWNED BY users.embeddings.id;


--
-- Name: tasks; Type: TABLE; Schema: users; Owner: postgres
--

CREATE TABLE users.tasks (
    id integer NOT NULL,
    task_id uuid DEFAULT gen_random_uuid() NOT NULL,
    source public.source_type NOT NULL,
    original_input text NOT NULL,
    status public.task_status DEFAULT 'pending'::public.task_status NOT NULL,
    error_message text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE users.tasks OWNER TO postgres;

--
-- Name: tasks_id_seq; Type: SEQUENCE; Schema: users; Owner: postgres
--

CREATE SEQUENCE users.tasks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE users.tasks_id_seq OWNER TO postgres;

--
-- Name: tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: users; Owner: postgres
--

ALTER SEQUENCE users.tasks_id_seq OWNED BY users.tasks.id;


--
-- Name: articles id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.articles ALTER COLUMN id SET DEFAULT nextval('public.articles_id_seq'::regclass);


--
-- Name: chunks id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chunks ALTER COLUMN id SET DEFAULT nextval('public.chunks_id_seq'::regclass);


--
-- Name: embeddings id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.embeddings ALTER COLUMN id SET DEFAULT nextval('public.embeddings_id_seq'::regclass);


--
-- Name: keywords id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.keywords ALTER COLUMN id SET DEFAULT nextval('public.keywords_id_seq'::regclass);


--
-- Name: models id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.models ALTER COLUMN id SET DEFAULT nextval('public.models_id_seq'::regclass);


--
-- Name: articles id; Type: DEFAULT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.articles ALTER COLUMN id SET DEFAULT nextval('users.articles_id_seq'::regclass);


--
-- Name: chunks id; Type: DEFAULT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.chunks ALTER COLUMN id SET DEFAULT nextval('users.chunks_id_seq'::regclass);


--
-- Name: embeddings id; Type: DEFAULT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.embeddings ALTER COLUMN id SET DEFAULT nextval('users.embeddings_id_seq'::regclass);


--
-- Name: tasks id; Type: DEFAULT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.tasks ALTER COLUMN id SET DEFAULT nextval('users.tasks_id_seq'::regclass);


--
-- Name: articles_keywords articles_keywords_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.articles_keywords
    ADD CONSTRAINT articles_keywords_pkey PRIMARY KEY (keyword_id, article_id);


--
-- Name: articles articles_md5_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.articles
    ADD CONSTRAINT articles_md5_key UNIQUE (md5);


--
-- Name: articles articles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.articles
    ADD CONSTRAINT articles_pkey PRIMARY KEY (id);


--
-- Name: chunks chunks_article_id_start_at_end_at_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chunks
    ADD CONSTRAINT chunks_article_id_start_at_end_at_key UNIQUE (article_id, start_at, end_at);


--
-- Name: chunks chunks_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chunks
    ADD CONSTRAINT chunks_pkey PRIMARY KEY (id);


--
-- Name: embeddings embeddings_article_id_chunk_id_model_id_vector_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.embeddings
    ADD CONSTRAINT embeddings_article_id_chunk_id_model_id_vector_key UNIQUE (article_id, chunk_id, model_id, vector);


--
-- Name: embeddings embeddings_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.embeddings
    ADD CONSTRAINT embeddings_pkey PRIMARY KEY (id);


--
-- Name: keywords keywords_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.keywords
    ADD CONSTRAINT keywords_pkey PRIMARY KEY (id);


--
-- Name: keywords keywords_term_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.keywords
    ADD CONSTRAINT keywords_term_key UNIQUE (term);


--
-- Name: models models_name_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.models
    ADD CONSTRAINT models_name_key UNIQUE (name);


--
-- Name: models models_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.models
    ADD CONSTRAINT models_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: articles_keywords articles_keywords_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.articles_keywords
    ADD CONSTRAINT articles_keywords_pkey PRIMARY KEY (keyword_id, article_id);


--
-- Name: articles articles_md5_key; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.articles
    ADD CONSTRAINT articles_md5_key UNIQUE (md5);


--
-- Name: articles articles_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.articles
    ADD CONSTRAINT articles_pkey PRIMARY KEY (id);


--
-- Name: chunks chunks_article_id_start_at_end_at_key; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.chunks
    ADD CONSTRAINT chunks_article_id_start_at_end_at_key UNIQUE (article_id, start_at, end_at);


--
-- Name: chunks chunks_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.chunks
    ADD CONSTRAINT chunks_pkey PRIMARY KEY (id);


--
-- Name: embeddings embeddings_article_id_chunk_id_model_id_vector_key; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.embeddings
    ADD CONSTRAINT embeddings_article_id_chunk_id_model_id_vector_key UNIQUE (article_id, chunk_id, model_id, vector);


--
-- Name: embeddings embeddings_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.embeddings
    ADD CONSTRAINT embeddings_pkey PRIMARY KEY (id);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);


--
-- Name: tasks tasks_task_id_key; Type: CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.tasks
    ADD CONSTRAINT tasks_task_id_key UNIQUE (task_id);


--
-- Name: articles_keywords articles_keywords_article_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.articles_keywords
    ADD CONSTRAINT articles_keywords_article_id_fkey FOREIGN KEY (article_id) REFERENCES public.articles(id) ON DELETE CASCADE;


--
-- Name: articles_keywords articles_keywords_keyword_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.articles_keywords
    ADD CONSTRAINT articles_keywords_keyword_id_fkey FOREIGN KEY (keyword_id) REFERENCES public.keywords(id) ON DELETE CASCADE;


--
-- Name: chunks chunks_article_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chunks
    ADD CONSTRAINT chunks_article_id_fkey FOREIGN KEY (article_id) REFERENCES public.articles(id) ON DELETE CASCADE;


--
-- Name: embeddings embeddings_article_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.embeddings
    ADD CONSTRAINT embeddings_article_id_fkey FOREIGN KEY (article_id) REFERENCES public.articles(id) ON DELETE CASCADE;


--
-- Name: embeddings embeddings_chunk_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.embeddings
    ADD CONSTRAINT embeddings_chunk_id_fkey FOREIGN KEY (chunk_id) REFERENCES public.chunks(id) ON DELETE CASCADE;


--
-- Name: embeddings embeddings_model_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.embeddings
    ADD CONSTRAINT embeddings_model_id_fkey FOREIGN KEY (model_id) REFERENCES public.models(id) ON DELETE CASCADE;


--
-- Name: articles_keywords articles_keywords_article_id_fkey; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.articles_keywords
    ADD CONSTRAINT articles_keywords_article_id_fkey FOREIGN KEY (article_id) REFERENCES users.articles(id) ON DELETE CASCADE;


--
-- Name: articles_keywords articles_keywords_keyword_id_fkey; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.articles_keywords
    ADD CONSTRAINT articles_keywords_keyword_id_fkey FOREIGN KEY (keyword_id) REFERENCES public.keywords(id) ON DELETE CASCADE;


--
-- Name: articles articles_task_id_fkey; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.articles
    ADD CONSTRAINT articles_task_id_fkey FOREIGN KEY (task_id) REFERENCES users.tasks(task_id) ON DELETE CASCADE;


--
-- Name: chunks chunks_article_id_fkey; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.chunks
    ADD CONSTRAINT chunks_article_id_fkey FOREIGN KEY (article_id) REFERENCES users.articles(id) ON DELETE CASCADE;


--
-- Name: embeddings embeddings_article_id_fkey; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.embeddings
    ADD CONSTRAINT embeddings_article_id_fkey FOREIGN KEY (article_id) REFERENCES users.articles(id) ON DELETE CASCADE;


--
-- Name: embeddings embeddings_chunk_id_fkey; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.embeddings
    ADD CONSTRAINT embeddings_chunk_id_fkey FOREIGN KEY (chunk_id) REFERENCES users.chunks(id) ON DELETE CASCADE;


--
-- Name: embeddings embeddings_model_id_fkey; Type: FK CONSTRAINT; Schema: users; Owner: postgres
--

ALTER TABLE ONLY users.embeddings
    ADD CONSTRAINT embeddings_model_id_fkey FOREIGN KEY (model_id) REFERENCES public.models(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

