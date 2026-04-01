package voiceingestapi_test

import (
	"testing"

	"github.com/casebrophy/planner/app/sdk/apitest"
)

func Test_VoiceIngest(t *testing.T) {
	t.Parallel()
	test := apitest.New(t, "Test_VoiceIngest")

	test.Run(t, ingest400(), "ingest-400")
	test.Run(t, ingest401(), "ingest-401")
}
