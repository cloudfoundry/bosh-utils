package redact_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-utils/redact"
)

const sensitive = "https://blob?X-Amz-Signature=supersecret"

var _ = Describe("Secret", func() {
	s := redact.Secret(sensitive)

	DescribeTable("redacts across every fmt verb",
		func(format string) {
			out := fmt.Sprintf(format, s)
			Expect(out).To(ContainSubstring(redact.Placeholder))
			Expect(out).ToNot(ContainSubstring("supersecret"))
		},
		Entry("%v", "%v"),
		Entry("%+v", "%+v"),
		Entry("%#v", "%#v"),
		Entry("%s", "%s"),
		Entry("%q", "%q"),
		Entry("%x", "%x"),
	)

	It("redacts via String and GoString", func() {
		Expect(s.String()).To(Equal(redact.Placeholder))
		Expect(s.GoString()).To(Equal(redact.Placeholder))
	})

	It("redacts inside a struct printed with %v", func() {
		wrapper := struct {
			ID  string
			URL redact.Secret
		}{ID: "id-1", URL: s}

		out := fmt.Sprintf("%v", wrapper)
		Expect(out).To(ContainSubstring("id-1"))
		Expect(out).To(ContainSubstring(redact.Placeholder))
		Expect(out).ToNot(ContainSubstring("supersecret"))
	})

	It("redacts via slog", func() {
		var buf strings.Builder
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		logger.Info("msg", "url", s)
		Expect(buf.String()).To(ContainSubstring(redact.Placeholder))
		Expect(buf.String()).ToNot(ContainSubstring("supersecret"))
	})

	It("reveals the real value only via Reveal", func() {
		Expect(s.Reveal()).To(Equal(sensitive))
	})

	It("round-trips the real value through JSON so functionality is preserved", func() {
		type doc struct {
			URL redact.Secret `json:"url"`
		}

		b, err := json.Marshal(doc{URL: s})
		Expect(err).ToNot(HaveOccurred())
		Expect(string(b)).To(ContainSubstring("supersecret"))

		var back doc
		Expect(json.Unmarshal(b, &back)).To(Succeed())
		Expect(back.URL.Reveal()).To(Equal(sensitive))
	})
})
