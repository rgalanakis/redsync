package rstest_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rgalanakis/redsync/rstest"
	"testing"
	"github.com/stvp/tempredis"
)

func TestRstest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rstest Suite")
}

var _ = Describe("redsync/rstest", func() {
	Describe("Servers", func() {
		It("can start and stop servers", func() {
			tr := make(rstest.Servers, 2)
			// Inits empty
			Expect(tr).To(HaveLen(2))
			Expect(tr[0]).To(BeNil())

			// Adds and starts servers
			tr.Start()
			Expect(tr[0]).To(BeAssignableToTypeOf(&tempredis.Server{}))

			// Manually stop the first server to show it won't error
			Expect(tr[0].Term()).To(Succeed())
			// Now stop all, and then stop the second, and show it errors
			tr.Stop()
			Expect(tr[0].Term()).To(MatchError("wait: no child processes"))
			Expect(tr[1].Term()).To(MatchError("wait: no child processes"))
		})
	})
})
