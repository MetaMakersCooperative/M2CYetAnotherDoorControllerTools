ALTER TABLE `member`
ADD `full_name` varchar(512);

UPDATE `member` SET `full_name` = CONCAT(`first_name`, " ", `last_name`);

ALTER TABLE `member`
DROP COLUMN `first_name`;
ALTER TABLE `member`
DROP COLUMN `last_name`;
