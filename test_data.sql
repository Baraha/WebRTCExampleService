DROP TABLE IF EXISTS videos CASCADE;

CREATE TABLE public.videos
(
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uri VARCHAR(100) NOT NULL
);

