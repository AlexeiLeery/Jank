package routes

import (
	"github.com/labstack/echo/v4"
	"jank.com/jank_blog/internal/middleware/render"
	"jank.com/jank_blog/pkg/serve/controller/post"
)

//func RegisterPostRoutes(r ...*echo.Group) {
//	// api v1 group
//	apiV1 := r[0]
//	postGroupV1 := apiV1.Group("/post")
//	postGroupV1.POST("/getOnePost", post.GetOnePost)
//	postGroupV1.GET("/getAllPosts", post.GetAllPosts)
//	postGroupV1.POST("/createOnePost", post.CreateOnePost, authMiddleware.JWTMiddleware())
//	postGroupV1.POST("/updateOnePost", post.UpdateOnePost, authMiddleware.JWTMiddleware())
//	postGroupV1.POST("/deleteOnePost", post.DeleteOnePost, authMiddleware.JWTMiddleware())
//}

func RegisterPostRoutes(r ...*echo.Group) {
	// api v1 group
	apiV1 := r[0]
	postGroupV1 := apiV1.Group("/post")
	postGroupV1.POST("/getOnePost", post.GetOnePost)
	postGroupV1.GET("/getAllPosts", post.GetAllPosts)
	postGroupV1.POST("/createOnePost", post.CreateOnePost, renderMiddleware.MarkdownRender())
	postGroupV1.POST("/updateOnePost", post.UpdateOnePost, renderMiddleware.MarkdownRender())
	postGroupV1.POST("/deleteOnePost", post.DeleteOnePost)
}
