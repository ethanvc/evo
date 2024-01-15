USE test_db;
CREATE TABLE user_tab(
                         user_id BIGINT PRIMARY KEY,
                         name VARCHAR(128) DEFAULT '',
                         status TINYINT DEFAULT 0,
                         gender TINYINT DEFAULT 0,
                         create_time BIGINT DEFAULT 0,
                         update_time BIGINT DEFAULT 0
)ENGINE=InnoDB;