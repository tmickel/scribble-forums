CREATE TABLE IF NOT EXISTS users (
   id INT GENERATED ALWAYS AS IDENTITY primary key,
   username varchar(128) not null,
   password varchar not null
);

CREATE UNIQUE INDEX username ON users (username);

CREATE TABLE IF NOT EXISTS sessions (
   id varchar primary key,
   user_id integer NOT NULL
);

CREATE TABLE IF NOT EXISTS topics (
   id INT GENERATED ALWAYS AS IDENTITY primary key,
   user_id integer NOT NULL,
   title varchar NOT NULL,
   latest_post_at timestamp NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
   id INT GENERATED ALWAYS AS IDENTITY primary key,
   user_id integer NOT NULL,
   topic_id integer NOT NULL,
   message varchar NOT NULL,
   created_at timestamp NOT NULL
);
