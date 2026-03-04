package bpmn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_SimpleProcess(t *testing.T) {
	// Given - simple BPMN process
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="simpleProcess" name="Simple Process" isExecutable="true">
    <startEvent id="start" name="Start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="userTask"/>
    <userTask id="userTask" name="User Task"/>
    <sequenceFlow id="flow2" sourceRef="userTask" targetRef="end"/>
    <endEvent id="end" name="End"/>
  </process>
</definitions>`)

	// When
	process, err := Parse(xml)

	// Then
	require.NoError(t, err, "Failed to parse BPMN")
	assert.NotNil(t, process, "Process should not be nil")
	assert.Equal(t, "simpleProcess", process.ID, "Process ID should match")
	assert.Equal(t, "Simple Process", process.Name, "Process name should match")
	assert.Equal(t, "true", process.IsExecutable, "Process should be executable")
	assert.Len(t, process.SequenceFlow, 2, "Should have two sequence flows")
}

func TestParse_InvalidXML(t *testing.T) {
	// Given
	invalidXML := []byte("<invalid><xml></invalid>")

	// When
	process, err := Parse(invalidXML)

	// Then
	assert.Error(t, err, "Should return error for invalid XML")
	assert.Nil(t, process, "Process should be nil for invalid XML")
}

func TestParse_MissingStartEvent(t *testing.T) {
	// Given - process without start event
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="noStartProcess" name="Process Without Start" isExecutable="true">
    <userTask id="userTask" name="User Task"/>
    <sequenceFlow id="flow1" sourceRef="userTask" targetRef="end"/>
    <endEvent id="end" name="End"/>
  </process>
</definitions>`)

	// When
	process, err := Parse(xml)

	// Then
	require.NoError(t, err, "Should parse process without start event")
	assert.Equal(t, "noStartProcess", process.ID)
}

func TestParse_ServiceTaskWithDelegate(t *testing.T) {
	// Given
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="serviceTaskProcess" name="Service Task Process" isExecutable="true">
    <startEvent id="start" name="Start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="serviceTask"/>
    <serviceTask id="serviceTask" name="Service Task" topic="test-topic"/>
    <sequenceFlow id="flow2" sourceRef="serviceTask" targetRef="end"/>
    <endEvent id="end" name="End"/>
  </process>
</definitions>`)

	// When
	process, err := Parse(xml)

	// Then
	require.NoError(t, err, "Failed to parse service task")
	assert.Equal(t, "serviceTaskProcess", process.ID)
}

func TestParse_GatewayWithConditions(t *testing.T) {
	// Given
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"
             xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <process id="gatewayProcess" name="Gateway Process" isExecutable="true">
    <startEvent id="start" name="Start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="gateway"/>
    
    <exclusiveGateway id="gateway" name="Exclusive Gateway"/>
    
    <sequenceFlow id="flow2" sourceRef="gateway" targetRef="task1">
      <conditionExpression xsi:type="tFormalExpression">${approved == true}</conditionExpression>
    </sequenceFlow>
    
    <userTask id="task1" name="Approved Task"/>
    <sequenceFlow id="flow3" sourceRef="task1" targetRef="end"/>
    
    <sequenceFlow id="flow4" sourceRef="gateway" targetRef="task2">
      <conditionExpression xsi:type="tFormalExpression">${approved == false}</conditionExpression>
    </sequenceFlow>
    
    <userTask id="task2" name="Rejected Task"/>
    <sequenceFlow id="flow5" sourceRef="task2" targetRef="end"/>
    
    <endEvent id="end" name="End"/>
  </process>
</definitions>`)

	// When
	process, err := Parse(xml)

	// Then
	require.NoError(t, err, "Failed to parse gateway")
	assert.Equal(t, "gatewayProcess", process.ID)
	assert.Len(t, process.SequenceFlow, 5, "Should have five sequence flows")

	var conditionCount int
	for _, flow := range process.SequenceFlow {
		if flow.ConditionExpression != nil && flow.ConditionExpression.Text != "" {
			conditionCount++
		}
	}
	assert.Equal(t, 2, conditionCount, "Should have two sequence flows with conditions")
}

func TestParseBytes(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="test" name="Test" isExecutable="true">
    <startEvent id="start"/>
    <endEvent id="end"/>
  </process>
</definitions>`)

	process, err := Parse(xml)
	require.NoError(t, err)
	assert.Equal(t, "test", process.ID)
}
