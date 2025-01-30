create table if not exists packages (   
  id uuid primary key default gen_random_uuid(),        
  name text not null,
  version text not null,
  architecture text not null,
  description text,
  maintainer text,
  depends text,
  path text not null,
  storage_url text not null,
  size bigint not null,
  md5 text not null,
  sha1 text not null,
  sha256 text not null,
  github_repo text not null,
  github_owner text not null,
  created_at timestamp with time zone default now(),    
  updated_at timestamp with time zone default now()     
);

create unique index if not exists packages_name_version_architecture_key on packages(name, version, architecture);
create index if not exists packages_github_owner_github_repo_idx on packages(github_owner, github_repo);
create index if not exists packages_name_idx on packages(name);
create index if not exists packages_path_idx on packages(path);

create table if not exists repository_metadata (
  id uuid primary key default gen_random_uuid(),
  type text not null unique,
  content text not null,
  updated_at timestamp with time zone default now()
);