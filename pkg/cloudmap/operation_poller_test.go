package cloudmap

import (
	"context"
	"testing"
	"time"

	cloudmapMock "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/mocks/pkg/cloudmap"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	op1      = "one"
	op2      = "two"
	op3      = "three"
	interval = 100 * time.Millisecond
	timeout  = 500 * time.Millisecond
)

func TestOperationPoller_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmapMock.NewMockServiceDiscoveryApi(mockController)

	op1First := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opSubmitted(), nil)
	op1Second := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opPending(), nil)
	op1Third := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opSuccess(), nil)
	gomock.InOrder(op1First, op1Second, op1Third)

	op2First := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opPending(), nil)
	op2Second := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opSuccess(), nil)
	gomock.InOrder(op2First, op2Second)

	sdApi.EXPECT().GetOperation(gomock.Any(), op3).Return(opSuccess(), nil)

	op := NewOperationPollerWithConfig(interval, timeout, sdApi)
	op.Submit(context.TODO(), op1)
	op.Submit(context.TODO(), op2)
	op.Submit(context.TODO(), op3)

	result := op.Await()
	assert.Nil(t, result)
}

func TestOperationPoller_AllFail(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmapMock.NewMockServiceDiscoveryApi(mockController)

	op1First := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opSubmitted(), nil)
	op1Second := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opPending(), nil)
	op1Third := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opFailed(), nil)
	gomock.InOrder(op1First, op1Second, op1Third)

	op2First := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opSubmitted(), nil)
	op2Second := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opFailed(), nil)
	gomock.InOrder(op2First, op2Second)

	op := NewOperationPollerWithConfig(interval, timeout, sdApi)
	op.Submit(context.TODO(), op1)
	op.Submit(context.TODO(), op2)

	err := op.Await()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), op1)
	assert.Contains(t, err.Error(), op2)
}

func TestOperationPoller_Mixed(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmapMock.NewMockServiceDiscoveryApi(mockController)

	op1First := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opSubmitted(), nil)
	op1Second := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opPending(), nil)
	op1Third := sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opFailed(), nil)
	gomock.InOrder(op1First, op1Second, op1Third)

	op2First := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opSubmitted(), nil)
	op2Second := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opPending(), nil)
	op2Third := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opSuccess(), nil)
	gomock.InOrder(op2First, op2Second, op2Third)

	op := NewOperationPollerWithConfig(interval, timeout, sdApi)
	op.Submit(context.TODO(), op1)
	op.Submit(context.TODO(), op2)

	err := op.Await()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), op1)
	assert.NotContains(t, err.Error(), op2)
}

func TestOperationPoller_Timeout(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmapMock.NewMockServiceDiscoveryApi(mockController)

	sdApi.EXPECT().GetOperation(gomock.Any(), op1).Return(opPending(), nil).AnyTimes()

	op2First := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opPending(), nil)
	op2Second := sdApi.EXPECT().GetOperation(gomock.Any(), op2).Return(opSuccess(), nil)
	gomock.InOrder(op2First, op2Second)

	op := NewOperationPollerWithConfig(interval, timeout, sdApi)
	op.Submit(context.TODO(), op1)
	op.Submit(context.TODO(), op2)

	err := op.Await()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), op1)
	assert.Contains(t, err.Error(), operationPollTimoutErrorMessage)
	assert.NotContains(t, err.Error(), op2)
}

func TestOperationPoller_Poll_HappyCase(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()

	sdApi := cloudmapMock.NewMockServiceDiscoveryApi(mockController)

	sdApi.EXPECT().GetOperation(context.TODO(), op1).Return(opPending(), nil)
	sdApi.EXPECT().GetOperation(context.TODO(), op1).Return(opSuccess(), nil)

	op := NewOperationPollerWithConfig(interval, timeout, sdApi)
	_, err := op.Poll(context.TODO(), op1)
	assert.Nil(t, err)
}

func opPending() *types.Operation {
	return &types.Operation{
		Status: types.OperationStatusPending,
	}
}

func opFailed() *types.Operation {
	return &types.Operation{
		Status:       types.OperationStatusFail,
		ErrorMessage: aws.String("fail"),
	}
}

func opSubmitted() *types.Operation {
	return &types.Operation{
		Status: types.OperationStatusSubmitted,
	}
}

func opSuccess() *types.Operation {
	return &types.Operation{
		Status: types.OperationStatusSuccess,
	}
}
