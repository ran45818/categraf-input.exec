CREATE TABLE `cmdb_vip_rserver` (
  `id` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `cmdb_id` bigint DEFAULT NULL COMMENT 'cmbd服务器id',
  `vip_port` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '虚拟IP与端口',
  `status` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '映射服务端口状态',
  `server` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '映射服务ip',
  `port` int DEFAULT NULL COMMENT '映射服务端口',
  `deleted` int DEFAULT '0' COMMENT '是否删除 1 是 0 否',
  `update_date` datetime DEFAULT NULL COMMENT '修改时间',
  `create_time` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `app_name` text,
  PRIMARY KEY (`id`),
  KEY `cmdb_vip_rserver_cmdb_id_IDX` (`cmdb_id`) USING BTREE,
  KEY `cmdb_vip_rserver_server_IDX` (`server`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='cmdb虚拟ip监听的服务器列表'
