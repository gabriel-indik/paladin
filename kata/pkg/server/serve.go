/*
 * Copyright © 2024 Kaleido, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
package server

import (
	"context"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/hyperledger/firefly-common/pkg/log"
	"google.golang.org/grpc"

	"github.com/kaleido-io/paladin/kata/internal/transaction"
	"github.com/kaleido-io/paladin/kata/pkg/proto"
)

type grpcServer struct {
	listener net.Listener
	server   *grpc.Server
	done     chan error
}

var serverLock sync.Mutex

var servers = map[string]*grpcServer{}

func newRPCServer(socketAddress string) (*grpcServer, error) {
	ctx := log.WithLogField(context.Background(), "pid", strconv.Itoa(os.Getpid()))
	log.L(ctx).Infof("server starting at unix socket %s", socketAddress)
	l, err := net.Listen("unix", socketAddress)
	if err != nil {
		log.L(ctx).Error("failed to listen: ", err)
		return nil, err
	}
	s := grpc.NewServer()

	proto.RegisterPaladinTransactionServiceServer(s, &transaction.PaladinTransactionService{})
	log.L(ctx).Infof("server listening at %v", l.Addr())
	return &grpcServer{
		listener: l,
		server:   s,
		done:     make(chan error),
	}, nil
}

func Run(ctx context.Context, socketAddress string) {
	serverLock.Lock()
	_, exists := servers[socketAddress]
	serverLock.Unlock()

	if exists {
		log.L(ctx).Errorf("Server %s already running", socketAddress)
		return
	}
	s, err := newRPCServer(socketAddress)
	if err != nil {
		return
	}

	serverLock.Lock()
	servers[socketAddress] = s
	serverLock.Unlock()

	log.L(ctx).Infof("Server %s started", socketAddress)
	s.done <- s.server.Serve(s.listener)
	log.L(ctx).Infof("Server %s ended", socketAddress)
}

func Stop(ctx context.Context, socketAddress string) {
	serverLock.Lock()
	s := servers[socketAddress]
	serverLock.Unlock()

	if s != nil {
		s.server.GracefulStop()
		serverErr := <-s.done
		log.L(ctx).Infof("Server %s stopped (err=%v)", socketAddress, serverErr)
	}

	serverLock.Lock()
	delete(servers, socketAddress)
	serverLock.Unlock()
}