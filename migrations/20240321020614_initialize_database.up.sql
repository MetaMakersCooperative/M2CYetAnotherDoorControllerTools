SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";

CREATE TABLE IF NOT EXISTS `accesscontrol` (
  `id` int NOT NULL AUTO_INCREMENT,
  `rfid_card_num` int NOT NULL,
  `rfid_card_val` int NOT NULL,
  `status` varchar(15) NOT NULL,
  `comment` varchar(80) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `rfid_card_num` (`rfid_card_num`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `member` (
  `id` int NOT NULL AUTO_INCREMENT,
  `member_num` int NOT NULL,
  `first_name` varchar(20) NOT NULL,
  `last_name` varchar(20) NOT NULL,
  `display_name` varchar(50) NOT NULL,
  `address` varchar(30) NOT NULL,
  `city` varchar(20) NOT NULL,
  `postal_code` varchar(6) NOT NULL,
  `province` varchar(2) NOT NULL,
  `email` varchar(40) NOT NULL,
  `m2c_email` varchar(30) NOT NULL,
  `phone_num` varchar(15) NOT NULL,
  `cellphone_flag` boolean NOT NULL,
  `bio` text,
  `skills` text,
  `interests` varchar(500) DEFAULT NULL,
  `verified_photo_id` boolean NOT NULL,
  `photo_id` mediumblob,
  `read_SOPs` boolean NOT NULL,
  `signed_waiver` boolean NOT NULL,
  `rfid_card_num` varchar(30) NOT NULL,
  `comments` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `member_num` (`member_num`),
  KEY `rfid_card_num` (`rfid_card_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `membermonth` (
  `id` int NOT NULL AUTO_INCREMENT,
  `month` int NOT NULL,
  `year` int(4) NOT NULL,
  `member_num` int NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `payment_num` int NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `date_member` (`year`,`month`,`member_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `payment` (
  `id` int NOT NULL AUTO_INCREMENT,
  `payment_num` int NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `payment_date` date NOT NULL,
  `bank_date` date DEFAULT NULL,
  `member_num` int NOT NULL,
  `payment_method` varchar(10) NOT NULL,
  `start_date` date DEFAULT NULL,
  `end_date` date NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `payment-num` (`payment_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE IF NOT EXISTS `receipt` (
  `id` int NOT NULL AUTO_INCREMENT,
  `receipt_num` int NOT NULL,
  `payment_id` int NOT NULL,
  `issue_date` int NOT NULL,
  `comment` text NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
