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

package grpctransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/kaleido-io/paladin/kata/pkg/proto"
	"github.com/kaleido-io/paladin/kata/pkg/proto/interpaladin"
	"github.com/stretchr/testify/assert"

	grpctransportpb "github.com/kaleido-io/paladin/kata/pkg/proto/grpctransport"
	interPaladinPB "github.com/kaleido-io/paladin/kata/pkg/proto/interpaladin"
)

type fakePayload struct {
	Key   string
	Value string
}

var (
	testPort        = 10002
	testBufferSize  = 1
	loopbackAddress = fmt.Sprintf("localhost:%d", testPort)
	sendingAddress  = fmt.Sprintf("localhost:%d", testPort+1)
	fakeDesintation = "somewhereoverthemoon"
)

type fakeExternalGRPCServer struct {
	interPaladinPB.UnimplementedInterPaladinTransportServer

	listener net.Listener
}

func (fegs *fakeExternalGRPCServer) SendInterPaladinMessage(ctx context.Context, message *interPaladinPB.InterPaladinMessage) (*interPaladinPB.InterPaladinMessage, error) {
	fegs.listener.Close()
	return nil, nil
}

func TestOutboundMessageFlowWithMTLS(t *testing.T) {
	ctx := context.Background()

	server1CaCertificate, err := os.ReadFile("../../../test/ca1/ca.crt")
	assert.NoError(t, err)
	server1ClientCertificate, err := tls.LoadX509KeyPair("../../../test/ca1/clients/client1.crt", "../../../test/ca1/clients/client1.key")
	assert.NoError(t, err)
	server1ServerCertificate, err := tls.LoadX509KeyPair("../../../test/ca1/clients/client2.crt", "../../../test/ca1/clients/client2.key")
	assert.NoError(t, err)

	server2CaCertificate, err := os.ReadFile("../../../test/ca2/ca.crt")
	assert.NoError(t, err)
	server2ServerCertificate, err := tls.LoadX509KeyPair("../../../test/ca2/clients/client1.crt", "../../../test/ca2/clients/client1.key")
	assert.NoError(t, err)

	server, err := NewExternalGRPCServer(ctx, testPort, testBufferSize, &server1ServerCertificate, &server1ClientCertificate)
	defer server.Shutdown()
	assert.NoError(t, err)

	// Start a server to recieve messages through
	testLis, err := net.Listen("tcp", fmt.Sprintf(":%d", testPort+1))
	assert.NoError(t, err)
	fakeServer := &fakeExternalGRPCServer{
		listener: testLis,
	}

	// Configure the server to listen to the message on ensure it's doing mTLS
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(server1CaCertificate))
	assert.Equal(t, true, ok)

	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{server2ServerCertificate},
		RootCAs:      certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})))
	defer s.GracefulStop()
	defer testLis.Close()
	interPaladinPB.RegisterInterPaladinTransportServer(s, fakeServer)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = s.Serve(testLis)
		wg.Done()
	}()

	// Create a fake message and queue it for sending
	fakeInternalMessage := &proto.Message{
		Id:          "some-uuid",
		Destination: fmt.Sprintf("localhost:%d", testPort+1),
	}
	fakeInternalMessagePb, err := anypb.New(fakeInternalMessage)
	assert.NoError(t, err)

	transportInformation := &grpctransportpb.GRPCTransportInformation{
		Address:       sendingAddress,
		CaCertificate: string(server2CaCertificate),
	}
	transportInformationPb, err := anypb.New(transportInformation)
	assert.NoError(t, err)

	fakeMessage := &proto.ExternalMessage{
		Body: fakeInternalMessagePb,
		TransportInformation: transportInformationPb,
	}

	server.QueueMessageForSend(fakeMessage)

	// Fake server will close when it gets a message
	wg.Wait()
}

func TestOutboundMessageFlow(t *testing.T) {
	ctx := context.Background()
	server, err := NewExternalGRPCServer(ctx, testPort, testBufferSize, nil, nil)
	defer server.Shutdown()
	assert.NoError(t, err)

	// Start a server to recieve messages through
	testLis, err := net.Listen("tcp", fmt.Sprintf(":%d", testPort+1))
	assert.NoError(t, err)
	fakeServer := &fakeExternalGRPCServer{
		listener: testLis,
	}
	s := grpc.NewServer()
	defer s.GracefulStop()
	defer testLis.Close()
	interPaladinPB.RegisterInterPaladinTransportServer(s, fakeServer)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = s.Serve(testLis)
		wg.Done()
	}()

	fakeInternalMessage := &proto.Message{
		Id:          "some-uuid",
		Destination: fmt.Sprintf("localhost:%d", testPort+1),
	}
	fakeInternalMessagePb, err := anypb.New(fakeInternalMessage)
	assert.NoError(t, err)

	transportInformation := &grpctransportpb.GRPCTransportInformation{
		Address:       sendingAddress,
	}
	transportInformationPb, err := anypb.New(transportInformation)
	assert.NoError(t, err)

	fakeMessage := &proto.ExternalMessage{
		Body: fakeInternalMessagePb,
		TransportInformation: transportInformationPb,
	}

	server.QueueMessageForSend(fakeMessage)
	wg.Wait()
}

func TestInboundMessageFlowWithMTLS(t *testing.T) {
	ctx := context.Background()

	// Create a server that has mTLS configured
	caCertificate, err := os.ReadFile("../../../test/ca1/ca.crt")
	assert.NoError(t, err)
	clientCertificate, err := tls.LoadX509KeyPair("../../../test/ca1/clients/client1.crt", "../../../test/ca1/clients/client1.key")
	assert.NoError(t, err)
	serverCertificate, err := tls.LoadX509KeyPair("../../../test/ca1/clients/client2.crt", "../../../test/ca1/clients/client2.key")
	assert.NoError(t, err)

	server, err := NewExternalGRPCServer(ctx, testPort, testBufferSize, &serverCertificate, &clientCertificate)
	defer server.Shutdown()
	assert.NoError(t, err)

	// Create a client that's able to do mTLS to the server
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(caCertificate))
	assert.Equal(t, true, ok)

	clientTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCertificate}, // just re-use the client cert for this test
		RootCAs:      certPool,
	}

	conn, err := grpc.NewClient(loopbackAddress, grpc.WithTransportCredentials(credentials.NewTLS(clientTLSConfig)))
	assert.NoError(t, err)
	defer conn.Close()

	client := interpaladin.NewInterPaladinTransportClient(conn)

	fakeInternalMessage := &proto.Message{
		Id:          "some-uuid",
		Destination: fakeDesintation,
	}

	fakeMessage := &ExternalMessage{ // TODO: Why is this not broken?
		Message:         *fakeInternalMessage,
		ExternalAddress: loopbackAddress,
		CACertificate:   string(caCertificate), // This is actually redundant for this specific test
	}

	mPay, err := anypb.New(fakeMessage)
	assert.NoError(t, err)

	// This flow is showing what happens when we get an inbound message, in this case we're going to pretend
	// we know about the Paladin that sent us the message, so we're going to manually add its CA into our store
	server.serverCertPool.AppendCertsFromPEM([]byte(caCertificate))

	_, err = client.SendInterPaladinMessage(ctx, &interpaladin.InterPaladinMessage{
		Body: mPay,
	})
	assert.NoError(t, err)

	recvMessageFlow := server.GetMessages(destination(fakeDesintation))

	msg := <-recvMessageFlow
	assert.NotNil(t, msg)
}

func TestInboundMessageFlow(t *testing.T) {
	ctx := context.Background()
	server, err := NewExternalGRPCServer(ctx, testPort, testBufferSize, nil, nil)
	defer server.Shutdown()
	assert.NoError(t, err)

	conn, err := grpc.NewClient(loopbackAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	client := interpaladin.NewInterPaladinTransportClient(conn)

	fakeInternalMessage := &proto.Message{
		Id:          "some-uuid",
		Destination: fakeDesintation,
	}

	fakeMessage := &ExternalMessage{ // TODO: Why is this not broken?
		Message:         *fakeInternalMessage,
		ExternalAddress: loopbackAddress,
	}

	mPay, err := anypb.New(fakeMessage)
	assert.NoError(t, err)

	_, err = client.SendInterPaladinMessage(ctx, &interpaladin.InterPaladinMessage{
		Body: mPay,
	})
	assert.NoError(t, err)

	recvMessageFlow := server.GetMessages(destination(fakeDesintation))

	msg := <-recvMessageFlow
	assert.NotNil(t, msg)
}

func TestInitializeExternalListener(t *testing.T) {
	ctx := context.Background()
	_, err := NewExternalGRPCServer(ctx, 10002, 1, nil, nil)
	assert.NoError(t, err)
}
