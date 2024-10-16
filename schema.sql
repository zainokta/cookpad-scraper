create table if not exists recipes(
	uuid uuid primary key default uuid_generate_v7(),
	recipe_id integer unique not null,
	title varchar(255) not null,
	link varchar(255) not null,
	image varchar(255) not null,
	created_at timestamptz default now(),
	updated_at timestamptz default now()
);
