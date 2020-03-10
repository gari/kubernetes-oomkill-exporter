package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/euank/go-kmsg-parser/kmsgparser"
	"github.com/stretchr/testify/require"
)

func TestGetPodInfoFromKube(t *testing.T) {
	klog, _, containerIDs := getTestData()

	// var extractedContainerIDs []string
	// var extractedPodUIDs []string
	for _, msg := range klog {
		parsedMsg, err := parseMessage(msg)
		require.NoError(t, err, "There should be no error while parsing kernel log")
		uid, cid := getContainerIDFromLog(parsedMsg.Message)
		fmt.Println(uid)
		fmt.Println(cid)
	}
	// fmt.Println("podUIDs:" + podUIDs)
	fmt.Println(containerIDs)
	// fmt.Println(t)
}

func TestGetPodUIDFromLog(t *testing.T) {
	klog, podUIDs, containerIDs := getTestDataKernel5()

	var extractedContainerIDs []string
	var extractedPodUIDs []string

	for _, msg := range klog {
		parsedMsg, err := parseMessage(msg)
		require.NoError(t, err, "There should be no error while parsing kernel log")
		uid, cid := getContainerIDFromLog(parsedMsg.Message)
		fmt.Println(uid)
		extractedContainerIDs = append(extractedContainerIDs, cid)
		extractedPodUIDs = append(extractedPodUIDs, uid)
	}

	require.Equal(t, containerIDs, extractedContainerIDs, "Extracted container ids do not match the expected result")
	require.Equal(t, podUIDs, extractedPodUIDs, "Extracted container ids do not match the expected result")
}

func TestGetPodUIDFromLogKernel5(t *testing.T) {
	klog, podUIDs, containerIDs := getTestData()

	var extractedContainerIDs []string
	var extractedPodUIDs []string

	for _, msg := range klog {
		parsedMsg, err := parseMessage(msg)
		require.NoError(t, err, "There should be no error while parsing kernel log")
		uid, cid := getContainerIDFromLog(parsedMsg.Message)
		fmt.Println(uid)
		extractedContainerIDs = append(extractedContainerIDs, cid)
		extractedPodUIDs = append(extractedPodUIDs, uid)
	}

	require.Equal(t, containerIDs, extractedContainerIDs, "Extracted container ids do not match the expected result")
	require.Equal(t, podUIDs, extractedPodUIDs, "Extracted container ids do not match the expected result")
}
func parseMessage(input string) (kmsgparser.Message, error) {
	// Format:
	//   PRIORITY,SEQUENCE_NUM,TIMESTAMP,-;MESSAGE
	parts := strings.SplitN(input, ";", 2)
	if len(parts) != 2 {
		return kmsgparser.Message{}, fmt.Errorf("invalid kmsg; must contain a ';'")
	}

	metadata, message := parts[0], parts[1]

	metadataParts := strings.Split(metadata, ",")
	if len(metadataParts) < 3 {
		return kmsgparser.Message{}, fmt.Errorf("invalid kmsg: must contain at least 3 ',' separated pieces at the start")
	}

	return kmsgparser.Message{
		Message: message,
	}, nil
}

func getTestData() ([]string, []string, []string) {
	return []string{
			"6,22743,6115623303887,-;Task in /kubepods/burstable/pode501ca8a-ec23-11e8-b17a-0a586444015a/f24766bce80e0ce4f0ca2887da2be9d0d250448d7ef503d9f85bf5e549c757d5 killed as a result of limit of /kubepods/burstable/pode501ca8a-ec23-11e8-b17a-0a586444015a",
			"6,23800,6780904484233,-;Task in /kubepods/burstable/pod0c4e2576-ef09-11e8-b17a-0a586444015a/9df959ad4292532c5d551226063bd840b906cbf118983fffefa0e3ab90331dc2 killed as a result of limit of /kubepods/burstable/pod0c4e2576-ef09-11e8-b17a-0a586444015a/9df959ad4292532c5d551226063bd840b906cbf118983fffefa0e3ab90331dc2",
		},
		[]string{
			"e501ca8a-ec23-11e8-b17a-0a586444015a",
			"0c4e2576-ef09-11e8-b17a-0a586444015a",
		},
		[]string{
			"f24766bce80e0ce4f0ca2887da2be9d0d250448d7ef503d9f85bf5e549c757d5",
			"9df959ad4292532c5d551226063bd840b906cbf118983fffefa0e3ab90331dc2",
		}
}

func getTestDataKernel5() ([]string, []string, []string) {
	return []string{
			"6,5779,2546245757643,-;oom-kill:constraint=CONSTRAINT_MEMCG,nodemask=(null),cpuset=6124b30c3ce132e718bf757061f154fff4d99794e5f609f8ce97da096d3973b1,mems_allowed=0-1,oom_memcg=/kubepods/burstable/pod396972f6-600a-11ea-8b76-047d7bae2198,task_memcg=/kubepods/burstable/pod396972f6-600a-11ea-8b76-047d7bae2198/6124b30c3ce132e718bf757061f154fff4d99794e5f609f8ce97da096d3973b1,task=gomemleak,pid=11420,uid=0",
		},
		[]string{
			"396972f6-600a-11ea-8b76-047d7bae2198",
		},
		[]string{
			"6124b30c3ce132e718bf757061f154fff4d99794e5f609f8ce97da096d3973b1",
		}
}
