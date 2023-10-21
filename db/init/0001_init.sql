CREATE USER shortener
    PASSWORD 'root';

CREATE DATABASE shortener
    OWNER 'shortener'
    ENCODING 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8';