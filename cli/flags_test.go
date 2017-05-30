/*
 * Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package cli_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-instance-logs-cli-plugin/cli"
)

var _ = Describe("Flags", func() {

	var (
		args           = []string{"cf", "sil", "my-service", "--recent"}
		recent         bool
		sslNoVerify    bool
		positionalArgs []string
		err            error
	)

	JustBeforeEach(func() {
		recent, sslNoVerify, positionalArgs, err = cli.ParseFlags(args)
	})

	Context("when an unexpected flag is received", func() {
		BeforeEach(func() {
			args = []string{"cf", "sil", "my-service", "-z"}
		})

		It("should raise a suitable error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("Error parsing arguments: Invalid flag: -z"))
		})
	})

	Describe("recent flag", func() {
		Context("when the recent flag is set", func() {
			BeforeEach(func() {
				args = []string{"cf", "sil", "my-service", "--recent"}
			})

			It("should capture the flag's value", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(recent).To(BeTrue())
			})
		})

		Context("when the recent flag is not set", func() {
			BeforeEach(func() {
				args = []string{"cf", "sil", "my-service"}
			})

			It("should capture the flag's value", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(recent).To(BeFalse())
			})
		})

	})

	Describe("skip ssl validation flag", func() {
		Context("when the skip ssl validation flag is set", func() {
			BeforeEach(func() {
				args = []string{"cf", "sil", "my-service", "--skip-ssl-validation"}
			})

			It("should capture the flag's value", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(sslNoVerify).To(BeTrue())
			})
		})
		
		Context("when the skip ssl validation flag is not set", func() {
			BeforeEach(func() {
				args = []string{"cf", "sil", "my-service"}
			})

			It("should capture the flag's value", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(sslNoVerify).To(BeFalse())
			})
		})
	})

	Describe("positional arguments", func() {
		Context("when positional arguments are provided", func() {
			BeforeEach(func() {
				args = []string{"cf", "sil", "my-service"}
			})

				It("should capture an array of positional arguments", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(len(positionalArgs)).To(Equal(3))
					Expect(positionalArgs[2]).To(Equal("my-service"))
				})
		})

		Context("when no positional arguments are provided", func() {
			BeforeEach(func() {
				args = []string{"cf", "sil"}
			})

			It("should capture an array of positional arguments", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(len(positionalArgs)).To(Equal(2))
			})
		})
	})
})
