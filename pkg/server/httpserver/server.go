// Copyright 2020 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package httpserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/log"
	"go.uber.org/zap"

	"github.com/chaos-mesh/chaosd/pkg/config"
	"github.com/chaos-mesh/chaosd/pkg/core"
	"github.com/chaos-mesh/chaosd/pkg/server/chaosd"
	"github.com/chaos-mesh/chaosd/pkg/server/utils"
	"github.com/chaos-mesh/chaosd/pkg/swaggerserver"
)

type httpServer struct {
	conf   *config.Config
	chaos  *chaosd.Server
	exp    core.ExperimentStore
	engine *gin.Engine
}

func NewServer(
	conf *config.Config,
	chaos *chaosd.Server,
	exp core.ExperimentStore,
) *httpServer {
	e := gin.Default()
	e.Use(utils.MWHandleErrors())

	return &httpServer{
		conf:   conf,
		chaos:  chaos,
		exp:    exp,
		engine: e,
	}
}

func Register(s *httpServer) {
	if s.conf.Platform != config.LocalPlatform {
		return
	}

	handler(s)

	go func() {
		addr := s.conf.Address()
		log.Debug("starting HTTP server", zap.String("address", addr))

		if err := s.engine.Run(addr); err != nil {
			log.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()
}

func handler(s *httpServer) {
	api := s.engine.Group("/api")
	{
		api.GET("/swagger/*any", swaggerserver.Handler())
	}

	attack := api.Group("/attack")
	{
		attack.POST("/process", s.createProcessAttack)
		attack.POST("/stress", s.createStressAttack)
		attack.POST("/network", s.createNetworkAttack)
		attack.POST("/disk", s.createDiskAttack)

		attack.DELETE("/:uid", s.recoverAttack)
	}
}

// @Summary Create process attack.
// @Description Create process attack.
// @Tags attack
// @Produce json
// @Param request body core.ProcessCommand true "Request body"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.APIError
// @Failure 500 {object} utils.APIError
// @Router /api/attack/process [post]
func (s *httpServer) createProcessAttack(c *gin.Context) {
	attack := &core.ProcessCommand{}
	if err := c.ShouldBindJSON(attack); err != nil {
		c.AbortWithError(http.StatusBadRequest, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	uid, err := s.chaos.ProcessAttack(attack)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	c.JSON(http.StatusOK, utils.AttackSuccessResponse(uid))
}

// @Summary Create network attack.
// @Description Create network attack.
// @Tags attack
// @Produce json
// @Param request body core.NetworkCommand true "Request body"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.APIError
// @Failure 500 {object} utils.APIError
// @Router /api/attack/network [post]
func (s *httpServer) createNetworkAttack(c *gin.Context) {
	attack := &core.NetworkCommand{}
	if err := c.ShouldBindJSON(attack); err != nil {
		c.AbortWithError(http.StatusBadRequest, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	uid, err := s.chaos.NetworkAttack(attack)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	c.JSON(http.StatusOK, utils.AttackSuccessResponse(uid))
}

// @Summary Create stress attack.
// @Description Create stress attack.
// @Tags attack
// @Produce json
// @Param request body core.StressCommand true "Request body"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.APIError
// @Failure 500 {object} utils.APIError
// @Router /api/attack/stress [post]
func (s *httpServer) createStressAttack(c *gin.Context) {
	attack := &core.StressCommand{}
	if err := c.ShouldBindJSON(attack); err != nil {
		c.AbortWithError(http.StatusBadRequest, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	uid, err := s.chaos.StressAttack(attack)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	c.JSON(http.StatusOK, utils.AttackSuccessResponse(uid))
}

// @Summary Create disk attack.
// @Description Create disk attack.
// @Tags attack
// @Produce json
// @Param request body core.DiskCommand true "Request body"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.APIError
// @Failure 500 {object} utils.APIError
// @Router /api/attack/disk [post]
func (s *httpServer) createDiskAttack(c *gin.Context) {
	attack := &core.DiskCommand{}
	if err := c.ShouldBindJSON(attack); err != nil {
		c.AbortWithError(http.StatusBadRequest, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	var uid string
	var err error
	switch attack.Action {
	case core.DiskFillAction:
		uid, err = s.chaos.DiskFill(attack)
	case core.DiskReadPayloadAction:
		uid, err = s.chaos.DiskPayload(attack)
	case core.DiskWritePayloadAction:
		uid, err = s.chaos.DiskPayload(attack)
	default:
		c.AbortWithError(http.StatusBadRequest,
			utils.ErrInvalidRequest.WrapWithNoMessage(fmt.Errorf("invalid disk attack action %v", attack.Action)))
		return
	}

	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	c.JSON(http.StatusOK, utils.AttackSuccessResponse(uid))
}

// @Summary Create recover attack.
// @Description Create recover attack.
// @Tags attack
// @Produce json
// @Param uid path string true "uid"
// @Success 200 {object} utils.Response
// @Failure 500 {object} utils.APIError
// @Router /api/attack/{uid} [delete]
func (s *httpServer) recoverAttack(c *gin.Context) {
	uid := c.Param("uid")
	err := utils.RecoverExp(s.exp, s.chaos, uid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, utils.ErrInternalServer.WrapWithNoMessage(err))
		return
	}

	c.JSON(http.StatusOK, utils.RecoverSuccessResponse(uid))
}
