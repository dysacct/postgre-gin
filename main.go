// 资产管理系统, 支持全局搜索、缓存、JWT认证
package main

import (
	"context"
	"fmt"
	"gin-postgre-project/config"
	"gin-postgre-project/database"
	"gin-postgre-project/handlers"
	"gin-postgre-project/middleware"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置
	config.LoadConfig()
	// 2. 连接数据库 + 自动迁移 + 初始化用户
	database.ConnectDB()
	// 3. 连接 Redis
	database.ConnectRedis()
	r := gin.Default()

	// 公开路由
	r.POST("/api/login", handlers.Login)

	// 需要登录的路由组
	auth := r.Group("/api")
	auth.Use(middleware.AuthRequired()) // 使用中间件, 需要登录
	{
		auth.GET("/ping", func(c *gin.Context) {
			username, _ := c.Get("username")
			c.JSON(200, gin.H{
				"message":  "pong",
				"login_as": username,
			})
		})

		// 管理员专属路由
		admin := auth.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/dashboard", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "管理员面板"})
			})
		}
		// 机器相关路由
		auth.GET("/machines", handlers.ListMachines)                                     // 获取机器列表
		auth.GET("/machines/search", handlers.SearchMachinesByIDC)                       // 根据机房查询机器
		auth.GET("/machines/search/manufacturer", handlers.SearchMachinesByManufacturer) // 根据制造商模糊查询机器
		auth.GET("/machine/:ipmi_ip", handlers.GetMachine)                               // 获取单个机器
		auth.POST("/machine", handlers.CreateMachine)                                    // 创建机器
		auth.PUT("/machine/:ipmi_ip", handlers.UpdateMachine)                            // 更新机器
		auth.PUT("/machines/:ipmi_ip/business", handlers.UpdateBusinessInfo)             // 更新机器业务信息
		auth.PUT("/machines/:ipmi_ip/machine-info", handlers.UpdateMachineInfo)          // 更新机器硬件信息
		auth.DELETE("/machine/:ipmi_ip", handlers.DeleteMachine)                         // 删除机器
		auth.DELETE("/machines/:ipmi_ip/business", handlers.DeleteBusinessInfo)          // 删除业务信息
		auth.DELETE("/machines/:ipmi_ip/machine-info", handlers.DeleteMachineInfo)       // 删除机器硬件信息

		// 网络信息相关路由
		auth.GET("/network-info", handlers.ListNetworkInfo)                          // 获取网络信息列表（支持多字段过滤）
		auth.GET("/network-info/search", handlers.SearchNetworkInfo)                 // 模糊搜索网络信息（支持多字段）
		auth.GET("/network-info/stats", handlers.GetNetworkInfoStats)                // 获取网络信息统计
		auth.GET("/network-info/id/:id", handlers.GetNetworkInfoByID)                // 根据ID获取单个网络信息
		auth.GET("/network-info/ipmi/:ipmi_ip", handlers.GetNetworkInfoByIPMI)       // 根据IPMI IP获取网络信息列表
		auth.GET("/network-info/zbx/:zbx_id", handlers.GetNetworkInfoByZbxID)        // 根据ZBX ID获取网络信息列表
		auth.GET("/network-info/idc/:idc_code", handlers.GetNetworkInfoByIDC)        // 根据IDC机房代码获取网络信息
		auth.POST("/network-info", handlers.CreateNetworkInfo)                       // 创建网络信息
		auth.PUT("/network-info/:id", handlers.UpdateNetworkInfo)                    // 根据ID更新网络信息
		auth.PUT("/network-info/ipv4/:ipv4_ip", handlers.UpdateNetworkInfoByIPv4)    // 根据IPv4地址更新网络信息
		auth.PUT("/network-info/ipmi/:ipmi_ip", handlers.UpdateNetworkInfoByIPMI)    // 根据IPMI IP批量更新网络信息
		auth.PUT("/network-info/zbx/:zbx_id", handlers.UpdateNetworkInfoByZbxID)     // 根据ZBX ID批量更新网络信息
		auth.DELETE("/network-info/:id", handlers.DeleteNetworkInfo)                 // 根据ID删除单个网络信息
		auth.DELETE("/network-info/ipv4/:ipv4_ip", handlers.DeleteNetworkInfoByIPv4) // 根据IPv4地址删除网络信息
		auth.DELETE("/network-info/ipmi/:ipmi_ip", handlers.DeleteNetworkInfoByIPMI) // 根据IPMI IP批量删除网络信息
		auth.DELETE("/network-info/zbx/:zbx_id", handlers.DeleteNetworkInfoByZbxID)  // 根据ZBX ID批量删除网络信息

		// 管理员专属网络信息路由（危险操作）
		admin.DELETE("/network-info/clear-all", handlers.ClearAllNetworkInfo)        // 清空所有网络信息
		admin.DELETE("/network-info/idc/:idc_code", handlers.DeleteNetworkInfoByIDC) // 根据IDC机房代码批量删除网络信息

		// IDC信息相关路由
		auth.GET("/idc_info", handlers.GetIDCInfo)               // 获取IDC信息（ZbxID、IPMIIP、SSHIP）
		admin.DELETE("/idc/:idc_code", handlers.DeleteIDCByCode) // 根据IDC机房代码删除整个机房的所有数据（危险操作）

		// 删除管理（归档30天）
		auth.POST("/deletion/machines", handlers.DeleteMachines)  // 删除机器（归档30天）
		auth.POST("/deletion/networks", handlers.DeleteNetworks)  // 删除网络信息（归档30天）
		auth.GET("/deletion/records", handlers.GetDeletedRecords) // 查询已删除记录

		// Excel 导出
		auth.GET("/machines/export", handlers.ExportMachines)           // 导出机器信息
		auth.GET("/network-info/export", handlers.ExportNetworkInfo)    // 导出网络信息
		auth.GET("/business-info/export", handlers.ExportBusinessInfo)  // 导出业务信息
		auth.GET("/idc_info/export", handlers.ExportIDCInfo)            // 导出SSH信息
		auth.GET("/deletion/records/export", handlers.ExportDeletedRecords) // 导出删除记录
	}

	srv := &http.Server{
		Addr:    ":34185",
		Handler: r,
	}

	go func() {
		fmt.Println("CMDB API 服务启动 -> 端口 34185")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("服务器启动失败:", err)
		}
	}()

	// 启动过期删除记录清理（每12小时执行一次）
	go func() {
		for {
			time.Sleep(12 * time.Hour)
			handlers.CleanupExpiredRecords()
		}
	}()

	// 优雅退出：捕获 Ctrl+C
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stop // 阻塞等待信号

	fmt.Println("\n正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	fmt.Println("服务已关闭，程序安全退出")
}

