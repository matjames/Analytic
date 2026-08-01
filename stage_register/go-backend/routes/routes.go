package routes

import (
	"go-backend/handlers"
	"go-backend/middleware"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine) {
	// Create API route group
	api := r.Group("/api")
	{
		// Public user endpoints (no auth required)
		api.POST("/users/register", handlers.RegisterUser)
		api.POST("/users/login", handlers.LoginUser)
		api.POST("/users/change-password", handlers.ChangePassword)

		// Public endpoints (no auth required)
		api.GET("/authority", handlers.ListAuthorities)
		api.GET("/level", handlers.ListLevels)
		api.GET("/ownership", handlers.ListOwnerships)
		api.GET("/documents", handlers.ListDocuments)
		api.GET("/documents/:id", handlers.GetDocument)
		api.GET("/documents/:id/download", handlers.DownloadDocument)
		api.GET("/adminunits", handlers.ListAdminUnits)
		api.GET("/adminunits/tree/public", handlers.GetAdminUnitsTreePublic)
		api.GET("/adminunits/districts/public", handlers.ListDistrictsPublic)
		api.GET("/adminlevel/public", handlers.ListAdminLevelsPublic)

		// StatGate Field Operations v1 Route Aliases
		v1Field := api.Group("/v1/field")
		{
			v1Field.GET("/agencies", handlers.ListAuthorities)
			v1Field.GET("/tiers", handlers.ListLevels)
			v1Field.GET("/contracts", handlers.ListOwnerships)
			v1Field.GET("/territories", handlers.ListAdminUnits)
			v1Field.GET("/agents/public", handlers.ListFacilitiesPublic)
			v1Field.GET("/agents/public/:id", handlers.GetFacilityPublic)
			v1Field.GET("/agents/summary/contract-by-tier", handlers.GetFacilityOwnershipByLevel)
			v1Field.GET("/agents/summary/contract-totals", handlers.GetFacilityOwnershipTotals)
			v1Field.GET("/agents/filters", handlers.GetFacilityFilterOptions)
		}

		// Public facility endpoints (no auth required)
		api.GET("/facilities/public", handlers.ListFacilitiesPublic)
		api.GET("/facilities/public/:id", handlers.GetFacilityPublic)
		api.GET("/facilities/public/export", handlers.ExportFacilitiesPublic)
		api.GET("/facilities/summary/ownership-by-level", handlers.GetFacilityOwnershipByLevel)
		api.GET("/facilities/summary/ownership-totals", handlers.GetFacilityOwnershipTotals)
		api.GET("/facilities/filters", handlers.GetFacilityFilterOptions)
		api.GET("/facilities/distribution/ownership", handlers.GetFacilityDistributionByOwnership)
		api.GET("/facilities/distribution/level", handlers.GetFacilityDistributionByLevel)
		api.GET("/facilities/distribution/authority", handlers.GetFacilityDistributionByAuthority)

		// MFL API integration endpoints (requires basic auth)
		api.GET("/mfl/facilities", middleware.BasicAuthRequired(), handlers.ListFacilitiesMfl)
		api.GET("/mfl/level", middleware.BasicAuthRequired(), handlers.ListLevelsMfl)
		api.GET("/mfl/authority", middleware.BasicAuthRequired(), handlers.ListAuthoritiesMfl)
		api.GET("/mfl/ownership", middleware.BasicAuthRequired(), handlers.ListOwnershipsMfl)
		api.GET("/mfl/adminunits", middleware.BasicAuthRequired(), handlers.ListAdminUnitsMfl)

		// Organization Units API (requires basic auth)
		api.GET("/orgunits", middleware.BasicAuthRequired(), handlers.ListOrgUnits)
		api.GET("/orgunits/tree", middleware.BasicAuthRequired(), handlers.GetOrgUnitsTree)
		api.GET("/orgunits/level/:level", middleware.BasicAuthRequired(), handlers.ListOrgUnitsByLevel)
		api.GET("/orgunits/district/:id/facilities", middleware.BasicAuthRequired(), handlers.GetDistrictFacilities)
		api.GET("/orgunits/subcounty/:id/facilities", middleware.BasicAuthRequired(), handlers.GetSubcountyFacilities)
		api.GET("/orgunits/:id", middleware.BasicAuthRequired(), handlers.GetOrgUnit)
		api.GET("/orgunits/:id/children", middleware.BasicAuthRequired(), handlers.ListOrgUnitsWithChildren)

		// Authenticated endpoints
		auth := api.Group("/")
		auth.Use(middleware.AuthRequired())
		{
			// User endpoints (require auth)
			auth.GET("/users/me", handlers.GetCurrentUser)
			auth.GET("/users", handlers.ListUsers)
			auth.POST("/users", handlers.CreateUser)
			auth.POST("/users/upload", handlers.UploadUsersCSV)
			auth.GET("/users/:id", handlers.GetUser)
			auth.PUT("/users/:id", handlers.UpdateUser)
			auth.POST("/users/:id/reset-password", handlers.ResetPassword)
			auth.DELETE("/users/:id", handlers.DeleteUser)

			// Legacy endpoint (for backward compatibility)
			auth.GET("/me", handlers.GetCurrentUser)

			auth.POST("/posts", handlers.CreatePost)
			auth.GET("/posts", handlers.ListPosts)
			auth.GET("/posts/:id", handlers.GetPost)
			auth.PUT("/posts/:id", handlers.UpdatePost)
			auth.DELETE("/posts/:id", handlers.DeletePost)

			// Facility endpoints (require auth)
			auth.GET("/facilities", handlers.ListFacilities)
			auth.GET("/facilities/export", handlers.ExportFacilities)
			auth.GET("/facilities/:id", handlers.GetFacility)
			auth.POST("/facilities", handlers.CreateFacility)
			auth.POST("/facilities/upload", handlers.UploadFacilities)
			auth.PUT("/facilities/:id", handlers.UpdateFacility)
			auth.DELETE("/facilities/:id", handlers.DeleteFacility)

			// Authority endpoints (require auth)
			auth.GET("/authority/:id", handlers.GetAuthority)
			auth.POST("/authority", handlers.CreateAuthority)
			auth.PUT("/authority/:id", handlers.UpdateAuthority)
			auth.DELETE("/authority/:id", handlers.DeleteAuthority)

			// Level endpoints (require auth)
			auth.GET("/level/:id", handlers.GetLevel)
			auth.POST("/level", handlers.CreateLevel)
			auth.PUT("/level/:id", handlers.UpdateLevel)
			auth.DELETE("/level/:id", handlers.DeleteLevel)

			// Ownership endpoints (require auth)
			auth.GET("/ownership/:id", handlers.GetOwnership)
			auth.POST("/ownership", handlers.CreateOwnership)
			auth.PUT("/ownership/:id", handlers.UpdateOwnership)
			auth.DELETE("/ownership/:id", handlers.DeleteOwnership)

			// Document endpoints (require auth)
			auth.POST("/documents", handlers.UploadDocument)
			auth.PUT("/documents/:id", handlers.UpdateDocument)
			auth.DELETE("/documents/:id", handlers.DeleteDocument)

			// Admin level endpoints (require auth)
			auth.GET("/adminlevel", handlers.ListAdminLevels)
			auth.POST("/adminlevel", handlers.CreateAdminLevel)
			auth.PUT("/adminlevel/:id", handlers.UpdateAdminLevel)
			auth.DELETE("/adminlevel/:id", handlers.DeleteAdminLevel)
			auth.POST("/adminlevel/reorder", handlers.ReorderAdminLevels)

			// Admin units endpoints (require auth)
			auth.GET("/adminunits/paged", handlers.ListAdminUnitsPaged)
			auth.GET("/adminunits/public", handlers.GetAdminUnitsPublic)
			auth.GET("/adminunits/tree", handlers.GetAdminUnitsTree)
			auth.POST("/adminunits", handlers.CreateAdminUnit)
			auth.PUT("/adminunits/:id", handlers.UpdateAdminUnit)
			auth.DELETE("/adminunits/:id", handlers.DeleteAdminUnit)
			auth.POST("/adminunits/:id/move", handlers.MoveAdminUnit)
			auth.GET("/adminunits/:uid/descendants", handlers.GetAdminUnitDescendants)
			auth.GET("/adminunits/:uid/ancestors", handlers.GetAdminUnitAncestors)

			// Request endpoints (require auth)
			auth.POST("/requests", handlers.CreateRequest)
			auth.GET("/requests", handlers.ListRequests)
			auth.GET("/requests/stats", handlers.GetRequestStats)
			auth.GET("/requests/facilities", handlers.GetFacilitiesForSelection)
			auth.GET("/requests/district-info", handlers.GetDistrictInfo)
			auth.GET("/requests/:id", handlers.GetRequestById)
			auth.GET("/requests/:id/documents/:docId", handlers.DownloadRequestDocument)
			auth.POST("/requests/:id/approve", handlers.ApproveRequest)
			auth.POST("/requests/:id/reject", handlers.RejectRequest)

			// Dashboard endpoints (require auth)
			auth.GET("/dashboard/stats", handlers.GetDashboardStats)
		}
	}
}
