package cmd

import (
	"StoreServer/api"
	"StoreServer/api/auth"
	"StoreServer/config"
	"StoreServer/job"
	"StoreServer/middleware"
	"StoreServer/models"
	"StoreServer/utils/logger"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupServer() {
	con := config.GetConfig()
	errs := make(chan error, 1)

	//connecting to db
	fmt.Println("Connecting to database...")
	job.DBConnect()
	defer job.Disconnect()
	// init collection
	println("Init collection...")
	models.InitExampleDB()
	models.InitProductDB()
	models.InitCategoryDB()
	models.InitOrderDB()
	models.InitAuthDB()
	models.InitPurchaseDB()
	models.InitSupplierDB()
	println("Init collection done!")

	s := SetHandler()
	{
		go func() {
			server := &http.Server{
				Addr:              ":" + con.ServerPort,
				Handler:           s,
				WriteTimeout:      time.Second * 30,
				IdleTimeout:       time.Second * 30,
				ReadHeaderTimeout: time.Minute,
			}
			logger.GetLogger().Info("Server running on port: " + server.Addr)

			errs <- server.ListenAndServe()
		}()
	}

	err := <-errs
	if err != nil {
		logger.GetLogger().Error(err.Error())
	}
}

func SetHandler() *gin.Engine {
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodDelete,
			http.MethodPut,
			http.MethodPatch,
			http.MethodOptions,
		},
		AllowHeaders:           []string{"Origin", "Authorization", "Content-Type", "token"},
		AllowBrowserExtensions: true,
		AllowWebSockets:        true,
		AllowFiles:             true,
	}))

	r.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	authRoute := r.Group("/auth")
	{
		authRoute.POST("/login", auth.Login)
	}

	// Example routes
	apiRoute := r.Group("/api")
	apiRoute.Use(middleware.CheckLogin())
	{
		apiRoute.POST("/example/", api.CreateExample)
		apiRoute.GET("/examples/", api.GetExample)
		apiRoute.POST("/examples/", api.CreateListExample)
		apiRoute.PUT("/examples/", api.UpdateExample)
		apiRoute.DELETE("/examples/:id", api.DeleteExample)
		// Product routes
		apiRoute.GET("/products/", api.GetProduct)
		apiRoute.POST("/products/", middleware.CheckRole("admin", "manager"), api.CreateProduct)
		apiRoute.PUT("/products/:id", middleware.CheckRole("admin", "manager"), api.UpdateProduct)
		apiRoute.DELETE("/products/:id", middleware.CheckRole("admin", "manager"), api.DeleteProduct)
		// Category routes
		apiRoute.GET("/categories/", api.GetCategory)
		apiRoute.POST("/categories/", middleware.CheckRole("admin", "manager"), api.CreateCategory)
		apiRoute.PUT("/categories/:id", middleware.CheckRole("admin", "manager"), api.UpdateCategory)
		apiRoute.DELETE("/categories/:id", middleware.CheckRole("admin", "manager"), api.DeleteCategory)
		//Order routes
		apiRoute.POST("/orders/", middleware.CheckRole("admin", "manager", "cashier"), api.CreateOrder)
		apiRoute.GET("/orders/", api.GetOrder)
		apiRoute.GET("/orders/:id", api.GetOrderByID)
		apiRoute.DELETE("/orders/:id", middleware.CheckRole("admin", "manager"), api.DeleteOrder)
		apiRoute.PUT("/orders/", middleware.CheckRole("admin", "manager"), api.UpdateStatusOrder)
		apiRoute.PUT("/orders/:id/status", middleware.CheckRole("admin", "manager"), api.UpdateStatusOrder)
		// User routes
		apiRoute.POST("/users/", middleware.CheckRole("admin"), api.CreateUser)
		apiRoute.PUT("/users/", middleware.CheckRole("admin"), api.UpdateUser)
		apiRoute.PUT("/users/:id", middleware.CheckRole("admin"), api.UpdateUser)
		apiRoute.DELETE("/users/:id", middleware.CheckRole("admin"), api.DeleteUser)
		apiRoute.GET("/users/", middleware.CheckRole("admin"), api.GetUser)
		// Received Order
		apiRoute.POST("/purchases/", middleware.CheckRole("admin", "manager"), api.CreatePurchase)
		apiRoute.GET("/purchases/", api.GetPurchase)
		apiRoute.GET("/purchases/:id", api.GetPurchaseByID)
		apiRoute.PUT("/purchases/:id", middleware.CheckRole("admin", "manager"), api.UpdatePurchase)
		apiRoute.PUT("/purchases/:id/approve", middleware.CheckRole("admin", "manager"), api.ApprovePurchase)
		apiRoute.PUT("/purchases/:id/reject", middleware.CheckRole("admin", "manager"), api.RejectPurchase)
		apiRoute.DELETE("/purchases/:id", middleware.CheckRole("admin", "manager"), api.DeletePurchase)
		// Supplier routes
		apiRoute.POST("/suppliers/", middleware.CheckRole("admin", "manager"), api.CreateSupplier)
		apiRoute.GET("/suppliers/", api.GetSupplier)
		apiRoute.PUT("/suppliers/:id", middleware.CheckRole("admin", "manager"), api.UpdateSupplier)
		apiRoute.DELETE("/suppliers/:id", middleware.CheckRole("admin", "manager"), api.DeleteSupplier)
	}
	return r
}

