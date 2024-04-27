CREATE TABLE Users (
    Id STRING(36) NOT NULL,
    Name STRING(MAX) NOT NULL,
    Email STRING(MAX) NOT NULL,
    CreatedAt               TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    UpdatedAt               TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true) 
) PRIMARY KEY (Id)