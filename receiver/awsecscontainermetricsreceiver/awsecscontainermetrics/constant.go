// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsecscontainermetrics

const (
	AttributeECSDockerName        = "aws.ecs.docker.name"
	AttributeECSCluster           = "aws.ecs.cluster.name"
	AttributeECSTaskARN           = "aws.ecs.task.arn"
	AttributeECSTaskID            = "aws.ecs.task.id"
	AttributeECSTaskFamily        = "aws.ecs.task.family"
	AttributeECSTaskRevision      = "aws.ecs.task.version"
	AttributeECSServiceName       = "aws.ecs.service.name"
	AttributeECSTaskPullStartedAt = "aws.ecs.task.pull_started_at"
	AttributeECSTaskPullStoppedAt = "aws.ecs.task.pull_stopped_at"
	AttributeECSTaskKnownStatus   = "aws.ecs.task.known_status"
	AttributeECSTaskLaunchType    = "aws.ecs.task.launch_type"
	AttributeContainerImageID     = "aws.ecs.container.image.id"
	AttributeContainerCreatedAt   = "aws.ecs.container.created_at"
	AttributeContainerStartedAt   = "aws.ecs.container.started_at"
	AttributeContainerFinishedAt  = "aws.ecs.container.finished_at"
	AttributeContainerKnownStatus = "aws.ecs.container.know_status"
	AttributeContainerExitCode    = "aws.ecs.container.exit_code"

	CPUsInVCpu = 1024
	BytesInMiB = 1024 * 1024

	TaskPrefix      = "ecs.task."
	ContainerPrefix = "container."

	EndpointEnvKey   = "ECS_CONTAINER_METADATA_URI_V4"
	TaskStatsPath    = "/task/stats"
	TaskMetadataPath = "/task"

	AttributeMemoryUsage    = "memory.usage"
	AttributeMemoryMaxUsage = "memory.usage.max"
	AttributeMemoryLimit    = "memory.usage.limit"
	AttributeMemoryReserved = "memory.reserved"
	AttributeMemoryUtilized = "memory.utilized"

	AttributeCPUTotalUsage      = "cpu.usage.total"
	AttributeCPUKernelModeUsage = "cpu.usage.kernelmode"
	AttributeCPUUserModeUsage   = "cpu.usage.usermode"
	AttributeCPUSystemUsage     = "cpu.usage.system"
	AttributeCPUCores           = "cpu.cores"
	AttributeCPUOnlines         = "cpu.onlines"
	AttributeCPUReserved        = "cpu.reserved"
	AttributeCPUUtilized        = "cpu.utilized"
	AttributeCPUUsageInVCPU     = "cpu.usage.vcpu"

	AttributeNetworkRateRx = "network.rate.rx"
	AttributeNetworkRateTx = "network.rate.tx"

	AttributeNetworkRxBytes   = "network.io.usage.rx_bytes"
	AttributeNetworkRxPackets = "network.io.usage.rx_packets"
	AttributeNetworkRxErrors  = "network.io.usage.rx_errors"
	AttributeNetworkRxDropped = "network.io.usage.rx_dropped"
	AttributeNetworkTxBytes   = "network.io.usage.tx_bytes"
	AttributeNetworkTxPackets = "network.io.usage.tx_packets"
	AttributeNetworkTxErrors  = "network.io.usage.tx_errors"
	AttributeNetworkTxDropped = "network.io.usage.tx_dropped"

	AttributeStorageRead  = "storage.read_bytes"
	AttributeStorageWrite = "storage.write_bytes"

	AttributeDuration = "duration"

	UnitBytes       = "Bytes"
	UnitMegaBytes   = "Megabytes"
	UnitNanoSecond  = "Nanoseconds"
	UnitBytesPerSec = "Bytes/Second"
	UnitCount       = "Count"
	UnitVCpu        = "vCPU"
	UnitPercent     = "Percent"
	UnitSecond      = "Seconds"
)
