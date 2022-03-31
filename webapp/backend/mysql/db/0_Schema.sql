SET SESSION sql_mode='TRADITIONAL,NO_AUTO_VALUE_ON_ZERO,ONLY_FULL_GROUP_BY';
DROP DATABASE IF EXISTS isubnb;
CREATE DATABASE isubnb;

CREATE TABLE isubnb.user
(
    id        BIGINT NOT NULL AUTO_INCREMENT,
    password  VARCHAR(255) NULL,
    nickname  VARCHAR(255) NULL,
    thumbnail MEDIUMBLOB NULL,
    PRIMARY KEY (id)
);

CREATE TABLE  isubnb.home
(
    id              BIGINT NOT NULL AUTO_INCREMENT,
    name            VARCHAR(255) NULL,
    address         VARCHAR(255) NULL,
    location        VARCHAR(255) NULL,
    max_people_num  INT NULL,
    description     TEXT NULL,
    catch_phrase    TEXT NULL,
    attribute       TEXT NULL,
    style           VARCHAR(255) NULL,
    price           INT NULL,
    photo_1         MEDIUMTEXT NULL,
    photo_2         MEDIUMTEXT NULL,
    photo_3         MEDIUMTEXT NULL,
    photo_4         MEDIUMTEXT NULL,
    photo_5         MEDIUMTEXT NULL,
    rate            DOUBLE NULL,
    owner_id        BIGINT NOT NULL,
    PRIMARY KEY (id),
    INDEX           home_user_id_fk_idx (owner_id ASC),
    CONSTRAINT home_user_id_fk
        FOREIGN KEY (owner_id)
            REFERENCES isubnb.user (id)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION
);

CREATE TABLE isubnb.reservation_home
(
    id               VARCHAR(255) NOT NULL,
    user_id          BIGINT NOT NULL,
    home_id          BIGINT NOT NULL,
    date             DATE NOT NULL,
    number_of_people INT NULL,
    is_deleted       BIT(1) NULL,
    PRIMARY KEY (`id`, `date`),
    INDEX            reservation_home_user_id_fk_idx (user_id ASC),
    INDEX            reservation_home_home_id_fk_idx (home_id ASC),
    CONSTRAINT reservation_home_user_id_fk
        FOREIGN KEY (user_id)
            REFERENCES isubnb.user (id)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION,
    CONSTRAINT reservation_home_home_id_fk
        FOREIGN KEY (home_id)
            REFERENCES isubnb.home (id)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION
);

CREATE TABLE isubnb.activity
(
    id              BIGINT NOT NULL AUTO_INCREMENT,
    name            VARCHAR(255) NULL,
    address         VARCHAR(255) NULL,
    location        VARCHAR(255) NULL,
    max_people_num  INT NULL,
    description     TEXT NULL,
    catch_phrase    TEXT NULL,
    attribute       TEXT NULL,
    category        VARCHAR(255) NULL,
    price           INT NULL,
    photo_1         MEDIUMTEXT NULL,
    photo_2         MEDIUMTEXT NULL,
    photo_3         MEDIUMTEXT NULL,
    photo_4         MEDIUMTEXT NULL,
    photo_5         MEDIUMTEXT NULL,
    rate            DOUBLE NULL,
    owner_id        BIGINT NOT NULL,
    PRIMARY KEY (id),
    INDEX           activity_user_id_fk_idx (owner_id ASC),
    CONSTRAINT activity_user_id_fk
        FOREIGN KEY (owner_id)
            REFERENCES isubnb.user (id)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION
);

CREATE TABLE  isubnb.config
(
    reservable_days INT NULL
);

CREATE TABLE isubnb.reservation_activity
(
    id               VARCHAR(255) NOT NULL,
    user_id          BIGINT NOT NULL,
    activity_id      BIGINT NOT NULL,
    date             DATE NULL,
    number_of_people INT NULL,
    is_deleted       BIT(1) NULL,
    PRIMARY KEY (id),
    INDEX            user_id_idx (user_id ASC),
    INDEX            reservation_activity_activity_id_fk_idx (activity_id ASC),
    CONSTRAINT reservation_activity_user_id_fk
        FOREIGN KEY (user_id)
            REFERENCES isubnb.user (id)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION,
    CONSTRAINT reservation_activity_activity_id_fk
        FOREIGN KEY (activity_id)
            REFERENCES isubnb.activity (id)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION
);