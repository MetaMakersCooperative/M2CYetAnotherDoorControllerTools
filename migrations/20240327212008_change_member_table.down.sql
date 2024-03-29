ALTER TABLE `member` ADD `first_name` varchar(255);
ALTER TABLE `member` ADD `last_name` varchar(255);

-- Not ideal but there really shouldn't be a case where
-- we'll need to roll back a migration save for development
UPDATE `member` SET
`first_name` = SUBSTRING_INDEX(`full_name`,' ',1),
`last_name` = SUBSTRING_INDEX(`full_name`,' ',-1); 

ALTER TABLE `member` DROP COLUMN `full_name`;
