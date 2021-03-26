package handler

import (
	"context"
	"net/http"

	"encoding/json"

	"github.com/TianqiuHuang/grpc-client-app/pd/fight"
	"github.com/TianqiuHuang/grpc-client-app/pkg/cache"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
)

var (
	foreverAdminStream fight.FightSvc_AdminClient
)

// InitAdminClient ...
func InitAdminClient(fightSvcClient fight.FightSvcClient) error {
	var err error
	foreverAdminStream, err = fightSvcClient.Admin(context.Background())
	if err != nil {
		return err
	}

	for {
		resp, err := foreverAdminStream.Recv()
		if err != nil {
			return err
		}
		for _, hero := range resp.Heros {
			klog.Info("hero: '%v' has been updated", hero)
			if err = cache.HeroStore.Update(hero); err != nil {
				return err
			}
		} // for
	} // for
}

// AdjustHero ...
func AdjustHero() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := foreverAdminStream.Send(&fight.AdminRequest{
			Type: fight.AdminRequest_ADJUST_HERO,
		})
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}

// CreateHero ...
func CreateHero() gin.HandlerFunc {
	return func(c *gin.Context) {
		var hero fight.Hero
		err := json.NewDecoder(c.Request.Body).Decode(&hero)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = foreverAdminStream.Send(&fight.AdminRequest{
			Heros: []*fight.Hero{&hero},
			Type:  fight.AdminRequest_CREATE_HERO,
		})
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	} // return func
}
