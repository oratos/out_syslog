package syslog_test

import (
	"time"

	"code.cloudfoundry.org/rfc5424"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/fluent-bit-out-syslog/pkg/syslog"
)

var _ = Describe("Out", func() {
	Context("message routing and structure", func() {
		It("adds structured data to syslog message", func() {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "kube-system",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"pod_name":       []byte("etcd-minikube"),
					"namespace_name": []byte("kube-system"),
					"host":           []byte("some-host"),
					"container_name": []byte("etcd"),
				},
			}

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceivedWithSD(
				[]rfc5424.StructuredData{
					{
						ID: "kubernetes@47450",
						Parameters: []rfc5424.SDParam{
							{
								Name:  "namespace_name",
								Value: "kube-system",
							},
							{
								Name:  "object_name",
								Value: "etcd-minikube",
							},
							{
								Name:  "container_name",
								Value: "etcd",
							},
							{
								Name:  "vm_id",
								Value: "some-host",
							},
						},
					},
				},
			)
		})

		It("adds labels as structured data to syslog message", func() {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "kube-system",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"labels": map[interface{}]interface{}{
						"component":                     []byte("kube-addon-manager"),
						"kubernetes.io/minikube-addons": []byte("addon-manager"),
						"version":                       []byte("v8.6"),
					},
					"pod_name":       []byte("etcd-minikube"),
					"namespace_name": []byte("kube-system"),
					"host":           []byte("some-host"),
					"container_name": []byte("etcd"),
				},
			}

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceivedWithSD(
				[]rfc5424.StructuredData{
					{
						ID: "kubernetes@47450",
						Parameters: []rfc5424.SDParam{
							{
								Name:  "component",
								Value: "kube-addon-manager",
							},
							{
								Name:  "version",
								Value: "v8.6",
							},
							{
								Name:  "kubernetes.io/minikube-addons",
								Value: "addon-manager",
							},
							{
								Name:  "namespace_name",
								Value: "kube-system",
							},
							{
								Name:  "object_name",
								Value: "etcd-minikube",
							},
							{
								Name:  "container_name",
								Value: "etcd",
							},
							{
								Name:  "vm_id",
								Value: "some-host",
							},
						},
					},
				},
			)
		})

		It("skips labels that are not string/[]byte type", func() {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "kube-system",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"labels": map[interface{}]interface{}{
						"component": []byte("kube-addon-manager"),
						123:         []byte("addon-manager"),
						"stuff":     "addon-manager",
					},
					"namespace_name": []byte("kube-system"),
					"pod_name":       []byte("etcd-minikube"),
					"container_name": []byte("etcd"),
				},
			}

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceivedWithSD(
				[]rfc5424.StructuredData{
					{
						ID: "kubernetes@47450",
						Parameters: []rfc5424.SDParam{
							{
								Name:  "component",
								Value: "kube-addon-manager",
							},
							{
								Name:  "namespace_name",
								Value: "kube-system",
							},
							{
								Name:  "object_name",
								Value: "etcd-minikube",
							},
							{
								Name:  "container_name",
								Value: "etcd",
							},
						},
					},
				},
			)
		})

		Context("sink state", func() {
			It("keeps track of last sent message time", func() {
				spySink := newSpySink()
				defer spySink.stop()
				s := syslog.Sink{
					Addr:      spySink.url(),
					Namespace: "ns1",
					Name:      "sink-name",
				}
				out := syslog.NewOut([]*syslog.Sink{&s}, nil)
				record := map[interface{}]interface{}{
					"log": []byte("some-log"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns1"),
						"pod_name":       []byte("pod-name"),
						"container_name": []byte("container-name"),
					},
				}

				out.Write(record, time.Unix(0, 0).UTC(), "k8s.event._ns1_")

				spySink.accept().Close()

				Eventually(func() int64 {
					states := out.SinkState()
					Expect(states).To(HaveLen(1))

					stat := states[0]
					Expect(stat.Namespace).To(Equal("ns1"))
					Expect(stat.Name).To(Equal("sink-name"))
					Expect(stat.Error).To(BeNil())
					return stat.LastSuccessfulSend.UnixNano()
				}).Should(BeNumerically(">", 0))
			})

			It("keeps track of cluster sinks states", func() {
				spySink := newSpySink()
				defer spySink.stop()
				s := syslog.Sink{
					Addr: spySink.url(),
					Name: "sink-name",
				}
				out := syslog.NewOut(nil, []*syslog.Sink{&s})
				record := map[interface{}]interface{}{
					"log": []byte("some-log"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns1"),
						"pod_name":       []byte("pod-name"),
						"container_name": []byte("container-name"),
					},
				}

				out.Write(record, time.Unix(0, 0).UTC(), "k8s.event._ns1_")

				spySink.accept().Close()

				Eventually(func() int64 {
					states := out.SinkState()
					Expect(states).To(HaveLen(1))

					stat := states[0]
					Expect(stat.Namespace).To(Equal(""))
					Expect(stat.Name).To(Equal("sink-name"))
					Expect(stat.Error).To(BeNil())
					return stat.LastSuccessfulSend.UnixNano()
				}).Should(BeNumerically(">", 0))
			})

			It("tracks the latest error", func() {
				s := syslog.Sink{
					Addr:      "127.0.0.1:12345",
					Namespace: "ns1",
					Name:      "sink-name",
				}
				out := syslog.NewOut(
					[]*syslog.Sink{&s},
					nil,
					syslog.WithWriteTimeout(200*time.Millisecond),
				)
				record := map[interface{}]interface{}{
					"log": []byte("some-log"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns1"),
						"pod_name":       []byte("pod-name"),
						"container_name": []byte("container-name"),
					},
				}

				out.Write(record, time.Unix(0, 0).UTC(), "k8s.event._ns1_")
				out.Write(record, time.Unix(0, 0).UTC(), "k8s.event._ns1_")
				out.Write(record, time.Unix(0, 0).UTC(), "k8s.event._ns1_")

				var sErr *syslog.SinkError
				Eventually(func() *syslog.SinkError {
					states := out.SinkState()
					Expect(states).To(HaveLen(1))
					stat := states[0]
					Expect(stat.Namespace).To(Equal("ns1"))
					Expect(stat.Name).To(Equal("sink-name"))
					sErr = stat.Error
					return stat.Error
				}).ShouldNot(BeNil())

				Expect(sErr.Msg).To(Equal("dial tcp 127.0.0.1:12345: connect: connection refused"))
				Expect(sErr.Timestamp).To(BeTemporally(">", time.Time{}))
				Expect(sErr.Timestamp).To(BeTemporally("<", time.Now(), time.Second))

				spySink := newSpySink("127.0.0.1:12345")
				defer spySink.stop()
				record2 := map[interface{}]interface{}{
					"log": []byte("some-log"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns1"),
						"pod_name":       []byte("pod-name"),
						"container_name": []byte("container-name"),
					},
				}
				out.Write(record2, time.Unix(0, 0).UTC(), "k8s.event._ns1_")
				spySink.accept().Close()

				Eventually(func() *syslog.SinkError {
					states := out.SinkState()
					Expect(states).To(HaveLen(1))

					stat := states[0]
					Expect(stat.Namespace).To(Equal("ns1"))
					Expect(stat.Name).To(Equal("sink-name"))
					return stat.Error
				}).Should(BeNil())
			})
		})

		It("includes event in the app name if set on the record", func() {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "ns1",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns1"),
					"pod_name":       []byte("pod-name"),
					"container_name": []byte("container-name"),
				},
			}

			out.Write(record, time.Unix(0, 0).UTC(), "k8s.event._ns1_")

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - k8s.event/ns1/pod-name/container-name - - [kubernetes@47450 namespace_name="ns1" object_name="pod-name" container_name="container-name"] some-log` + "\n",
			)
		})

		It("doesn't return an error if it fails to write to one of the sinks in a namespace", func() {
			spySink1 := newSpySink()
			spySink1.stop()
			spySink2 := newSpySink()
			defer spySink2.stop()

			s1 := &syslog.Sink{
				Addr:      spySink1.url(),
				Namespace: "ns1",
			}
			s2 := &syslog.Sink{
				Addr:      spySink2.url(),
				Namespace: "ns1",
			}
			out := syslog.NewOut([]*syslog.Sink{s1, s2}, nil)

			r := map[interface{}]interface{}{
				"log": []byte("some-log-for-ns1"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns1"),
				},
			}
			out.Write(r, time.Unix(0, 0).UTC(), "pod.log")

			spySink2.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
			)
		})

		It("filters messages based on namespace_name in kubernetes metadata", func() {
			spySink := newSpySink()
			defer spySink.stop()

			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "kube-system",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"pod_name":       []byte("etcd-minikube"),
					"namespace_name": []byte("kube-system"),
					"host":           []byte("some-host"),
					"container_name": []byte("etcd"),
				},
			}

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/kube-system/etcd-minikube/etcd - - [kubernetes@47450 namespace_name="kube-system" object_name="etcd-minikube" container_name="etcd" vm_id="some-host"] some-log` + "\n",
			)
		})

		It("drops messages with unconfigured namespaces", func() {
			spySink := newSpySink()
			defer spySink.stop()

			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "test-namespace",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			r1 := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"pod_name":       []byte("etcd-minikube"),
					"namespace_name": []byte("kube-system"),
					"host":           []byte("some-host"),
					"container_name": []byte("etcd"),
				},
			}
			r2 := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"pod_name":       []byte("etcd-minikube"),
					"namespace_name": []byte("test-namespace"),
					"host":           []byte("some-host"),
					"container_name": []byte("etcd"),
				},
			}

			out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/test-namespace/etcd-minikube/etcd - - [kubernetes@47450 namespace_name="test-namespace" object_name="etcd-minikube" container_name="etcd" vm_id="some-host"] some-log` + "\n",
			)
		})

		It("truncates the app name if there is too much information", func() {
			spySink := newSpySink()
			defer spySink.stop()

			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "namespace-name-very-long",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"pod_name":       []byte("pod-name"),
					"namespace_name": []byte("namespace-name-very-long"),
					"host":           []byte("some-host"),
					"container_name": []byte("container-name-very-long"),
				},
			}

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/namespace-name-very-long/pod-name/contai - - [kubernetes@47450 namespace_name="namespace-name-very-long" object_name="pod-name" container_name="container-name-very-long" vm_id="some-host"] some-log` + "\n",
			)
		})

		It("doesn't add a newline if one already exists in the message", func() {
			spySink := newSpySink()
			defer spySink.stop()

			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "namespace",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			record := map[interface{}]interface{}{
				"log": []byte("some-log\n"),
				"kubernetes": map[interface{}]interface{}{
					"pod_name":       []byte("pod-name"),
					"namespace_name": []byte("namespace"),
					"host":           []byte("some-host"),
					"container_name": []byte("container"),
				},
			}

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/namespace/pod-name/container - - [kubernetes@47450 namespace_name="namespace" object_name="pod-name" container_name="container" vm_id="some-host"] some-log` + "\n",
			)
		})

		It("filters messages to multiple sinks for a namespace", func() {
			spySink1 := newSpySink()
			defer spySink1.stop()
			spySink2 := newTLSSpySink()
			defer spySink2.stop()

			s1 := &syslog.Sink{
				Addr:      spySink1.url(),
				Namespace: "ns1",
			}
			s2 := &syslog.Sink{
				Addr:      spySink2.url(),
				Namespace: "ns1",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}
			out := syslog.NewOut([]*syslog.Sink{s1, s2}, nil)

			r1 := map[interface{}]interface{}{
				"log": []byte("some-log-for-ns1"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns1"),
				},
			}
			go func() {
				defer GinkgoRecover()
				out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			}()

			spySink1.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
			)
			spySink2.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
			)
		})

		It("filters messages to multiple namespaces", func() {
			spySink1 := newSpySink()
			defer spySink1.stop()
			spySink2 := newSpySink()
			defer spySink2.stop()

			s1 := &syslog.Sink{
				Addr:      spySink1.url(),
				Namespace: "ns1",
			}
			s2 := &syslog.Sink{
				Addr:      spySink2.url(),
				Namespace: "ns2",
			}
			out := syslog.NewOut([]*syslog.Sink{s1, s2}, nil)

			r1 := map[interface{}]interface{}{
				"log": []byte("some-log-for-ns1"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns1"),
				},
			}
			r2 := map[interface{}]interface{}{
				"log": []byte("some-log-for-ns2"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns2"),
				},
			}
			out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")

			spySink1.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
			)
			spySink2.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns2// - - [kubernetes@47450 namespace_name="ns2" object_name="" container_name=""] some-log-for-ns2` + "\n",
			)
		})

		It("sends all messages to all cluster sinks", func() {
			spySink1 := newSpySink()
			defer spySink1.stop()
			spyClusterSink1 := newSpySink()
			defer spyClusterSink1.stop()
			spyClusterSink2 := newSpySink()
			defer spyClusterSink2.stop()

			s1 := &syslog.Sink{
				Addr:      spySink1.url(),
				Namespace: "ns1",
			}
			cs1 := &syslog.Sink{
				Addr: spyClusterSink1.url(),
			}
			cs2 := &syslog.Sink{
				Addr: spyClusterSink2.url(),
			}
			out := syslog.NewOut(
				[]*syslog.Sink{s1},
				[]*syslog.Sink{cs1, cs2},
			)

			r1 := map[interface{}]interface{}{
				"log": []byte("some-log-for-ns1"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns1"),
				},
			}
			r2 := map[interface{}]interface{}{
				"log": []byte("some-log-for-ns2"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns2"),
				},
			}
			out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")

			spySink1.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
			)
			spyClusterSink1.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1`+"\n",
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns2// - - [kubernetes@47450 namespace_name="ns2" object_name="" container_name=""] some-log-for-ns2`+"\n",
			)
			spyClusterSink2.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1`+"\n",
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns2// - - [kubernetes@47450 namespace_name="ns2" object_name="" container_name=""] some-log-for-ns2`+"\n",
			)
		})

	})

	It("uses the cluster_name as host if provided", func() {
		spySink := newSpySink()
		defer spySink.stop()
		s := syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "ns1",
		}
		out := syslog.NewOut([]*syslog.Sink{&s}, nil)
		record := map[interface{}]interface{}{
			"cluster_name": []byte("my-host"),
			"log":          []byte("some-log"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
				"pod_name":       []byte("pod-name"),
				"container_name": []byte("container-name"),
				"host":           []byte("some-host"),
			},
		}

		out.Write(record, time.Unix(0, 0).UTC(), "k8s.event")

		spySink.expectReceived(
			`<14>1 1970-01-01T00:00:00+00:00 my-host k8s.event/ns1/pod-name/container-name - - [kubernetes@47450 namespace_name="ns1" object_name="pod-name" container_name="container-name" vm_id="some-host"] some-log` + "\n",
		)
	})

	It("tracks messages dropped when trying to write to a full queue", func() {
		spySlowSink := newSpySink()
		defer spySlowSink.stop()

		slowSink := &syslog.Sink{
			Addr:      spySlowSink.url(),
			Namespace: "ns1",
		}
		out := syslog.NewOut(
			[]*syslog.Sink{slowSink},
			[]*syslog.Sink{},
			syslog.WithBufferSize(0),
		)

		numMessages := 10000

		writeDone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(writeDone)
			for i := 0; i < numMessages; i++ {
				r1 := map[interface{}]interface{}{
					"log": []byte("some-log-for-ns1"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns1"),
					},
				}
				out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			}
		}()

		select {
		case <-time.After(2 * time.Second):
			Fail("did not complete writes within two seconds")
		case <-writeDone:
		}

		Expect(slowSink.MessagesDropped()).To(BeNumerically(">=", 8000))
	})

	It("drops messages that exceed write deadline", func() {
		spySlowSink := newSpySink()
		defer spySlowSink.stop()

		slowSink := &syslog.Sink{
			Addr:      spySlowSink.url(),
			Namespace: "ns1",
		}
		out := syslog.NewOut(
			[]*syslog.Sink{slowSink},
			[]*syslog.Sink{},
			syslog.WithWriteTimeout(time.Nanosecond),
		)

		numMessages := 100
		writeDone := make(chan struct{})
		go func() {
			defer close(writeDone)
			for i := 0; i < numMessages; i++ {
				r1 := map[interface{}]interface{}{
					"log": []byte("some-log-for-ns1"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns1"),
					},
				}
				out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			}
		}()

		select {
		case <-time.After(2 * time.Second):
			Fail("did not complete writes within two seconds")
		case <-writeDone:
		}
		Eventually(slowSink.MessagesDropped, 2*time.Second).Should(BeNumerically("==", 100))
	})

	It("doesn't slow down a sink even if another sink isn't able to connect", func() {
		spySink := newSpySink()
		defer spySink.stop()

		sink := &syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "ns1",
		}

		badSink := &syslog.Sink{
			Addr: "example.com:63332",
		}
		out := syslog.NewOut(
			[]*syslog.Sink{sink},
			[]*syslog.Sink{badSink},
		)

		r1 := map[interface{}]interface{}{
			"log": []byte("some-log-for-ns1"),
			"kubernetes": map[interface{}]interface{}{
				"namespace_name": []byte("ns1"),
			},
		}

		writeDone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(writeDone)
			out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
		}()

		select {
		case <-time.After(2 * time.Second):
			Fail("did not complete writes within two seconds")
		case <-writeDone:
		}

		readDone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(readDone)
			spySink.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n",
			)
		}()

		select {
		case <-time.After(2 * time.Second):
			Fail("did not complete reads within two seconds")
		case <-readDone:
		}
	})

	It("continues to send messages to sinks even if one sink reads slowly", func() {
		spySink := newSpySink()
		defer spySink.stop()
		spySlowSink := newSpySink()
		defer spySlowSink.stop()

		sink := &syslog.Sink{
			Addr:      spySink.url(),
			Namespace: "ns1",
		}

		slowSink := &syslog.Sink{
			Addr: spySlowSink.url(),
		}
		out := syslog.NewOut(
			[]*syslog.Sink{sink},
			[]*syslog.Sink{slowSink},
		)

		numMessages := 10000

		readDone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(readDone)
			const expected = `<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns1// - - [kubernetes@47450 namespace_name="ns1" object_name="" container_name=""] some-log-for-ns1` + "\n"
			expectedMsgs := make([]string, 0, numMessages)
			for i := 0; i < numMessages; i++ {
				expectedMsgs = append(expectedMsgs, expected)
			}
			spySink.expectReceivedOnly(expectedMsgs...)
		}()

		writeDone := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(writeDone)
			for i := 0; i < numMessages; i++ {
				r1 := map[interface{}]interface{}{
					"log": []byte("some-log-for-ns1"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns1"),
					},
				}
				out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			}
		}()

		select {
		case <-time.After(5 * time.Second):
			Fail("did not complete writes within five seconds")
		case <-writeDone:
		}

		select {
		case <-time.After(5 * time.Second):
			Fail("did not complete reads within five seconds")
		case <-readDone:
		}
	})

	DescribeTable(
		"sends no data when record excludes pertinent info",
		func(record map[interface{}]interface{}) {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			done := make(chan struct{})
			go func() {
				_, _ = spySink.lis.Accept()
				close(done)
			}()

			Consistently(done).ShouldNot(BeClosed())
		},
		Entry(
			"has no k8s map",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
			},
		),
		Entry(
			"has k8s map of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[string][]byte{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
		),
		Entry(
			"has no namespace name",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
		),
		Entry(
			"has namespace of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []int{1, 2, 3, 4},
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
		),
	)

	DescribeTable(
		"prints even with some missing data",
		func(record map[interface{}]interface{}, message string) {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceived(message)
		},
		Entry(
			"no log message",
			map[interface{}]interface{}{
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container" vm_id="some-host"] `+"\n",
		),
		Entry(
			"log message is of different type",
			map[interface{}]interface{}{
				"log": []int{1, 2, 3, 4},
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container" vm_id="some-host"] `+"\n",
		),
		Entry(
			"log message key is of different type",
			map[interface{}]interface{}{
				5: []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container" vm_id="some-host"] `+"\n",
		),
		Entry(
			"no host",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] some-log`+"\n",
		),
		Entry(
			"host key is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					1:                []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] some-log`+"\n",
		),
		Entry(
			"host is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []int{1, 2, 3, 4},
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns/some-pod/some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="some-container"] some-log`+"\n",
		),
		Entry(
			"no container name",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/ - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="" vm_id="some-host"] some-log`+"\n",
		),
		Entry(
			"container name is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []byte("some-pod"),
					"container_name": []int{1, 2, 3, 4},
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns/some-pod/ - - [kubernetes@47450 namespace_name="some-ns" object_name="some-pod" container_name="" vm_id="some-host"] some-log`+"\n",
		),
		Entry(
			"no pod name",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns//some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="" container_name="some-container" vm_id="some-host"] some-log`+"\n",
		),
		Entry(
			"pod name is of different type",
			map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"host":           []byte("some-host"),
					"namespace_name": []byte("some-ns"),
					"pod_name":       []int{1, 2, 3, 4},
					"container_name": []byte("some-container"),
				},
			},
			`<14>1 1970-01-01T00:00:00+00:00 some-host pod.log/some-ns//some-container - - [kubernetes@47450 namespace_name="some-ns" object_name="" container_name="some-container" vm_id="some-host"] some-log`+"\n",
		),
	)

	Context("TCP", func() {
		It("eventually connects to a failing syslog sink", func() {
			spySink := newSpySink()
			spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "ns-123",
			}
			out := syslog.NewOut(
				[]*syslog.Sink{&s},
				nil,
				syslog.WithWriteTimeout(100*time.Millisecond),
			)
			f := func() int64 {
				r1 := map[interface{}]interface{}{
					"log": []byte("some-message"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("ns-123"),
					},
				}
				out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
				time.Sleep(10 * time.Millisecond)
				return s.MessagesDropped()
			}
			Eventually(f, 10*time.Second).Should(BeNumerically(">=", 1))
			// bring the sink back to life
			spySink = newSpySink(spySink.url())
			defer spySink.stop()

			r2 := map[interface{}]interface{}{
				"log": []byte("some-message-2"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("ns-123"),
				},
			}
			out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
			spySink.expectReceivedIncludes(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/ns-123// - - [kubernetes@47450 namespace_name="ns-123" object_name="" container_name=""] some-message-2` + "\n",
			)
		})

		It("doesn't reconnect if connection already established", func() {
			spySink := newSpySink()
			defer spySink.stop()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)

			record := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}
			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			out.Write(record, time.Unix(0, 0).UTC(), "pod.log")

			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				_, _ = spySink.lis.Accept()
			}()
			Consistently(done).ShouldNot(BeClosed())
		})

		It("reconnects if previous connection went away", func() {
			spySink := newSpySink()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			r1 := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			spySink.stop()

			// Write multiple messages because the first few msgs goes through
			// (not sure why) and eventually one gets dropped.
			f := func() int64 {
				r2 := map[interface{}]interface{}{
					"log": []byte("some-log-message-2"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("some-namespace"),
					},
				}
				out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
				return s.MessagesDropped()
			}
			Eventually(f).Should(BeNumerically(">=", 1))

			spySink = newSpySink(spySink.url())
			r3 := map[interface{}]interface{}{
				"log": []byte("some-log-message-3"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}
			out.Write(r3, time.Unix(0, 0).UTC(), "pod.log")

			spySink.expectReceivedIncludes(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message-3` + "\n",
			)
		})

	})

	Context("TLS", func() {
		It("eventually connects to a failing syslog sink", func() {
			spySink := newTLSSpySink()
			spySink.stop()
			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}
			out := syslog.NewOut(
				[]*syslog.Sink{s},
				nil,
				syslog.WithWriteTimeout(100*time.Millisecond),
			)
			timeToWait := 10 * time.Second
			goOn := make(chan struct{})
			go func() {
				defer GinkgoRecover()

				f := func() int64 {
					r1 := map[interface{}]interface{}{
						"log": []byte("some-log-message"),
						"kubernetes": map[interface{}]interface{}{
							"namespace_name": []byte("some-namespace"),
						},
					}
					out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
					time.Sleep(10 * time.Millisecond)
					return s.MessagesDropped()
				}
				Eventually(f, timeToWait).Should(BeNumerically(">=", 1))
				close(goOn)
			}()

			select {
			case <-goOn:
			case <-time.After(timeToWait):
				Fail("Unable to verify dropped tls message")
			}
			// bring the sink back to life
			spySink = newTLSSpySink(spySink.url())
			defer spySink.stop()

			go func() {
				r2 := map[interface{}]interface{}{
					"log": []byte("some-log-message-2"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("some-namespace"),
					},
				}
				out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
			}()

			spySink.expectReceivedIncludes(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message-2` + "\n",
			)
		})

		It("doesn't reconnect if connection already established", func() {
			spySink := newTLSSpySink()
			defer spySink.stop()
			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}
			out := syslog.NewOut([]*syslog.Sink{s}, nil)

			record := map[interface{}]interface{}{
				"log": []byte("some-log-message"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-namespace"),
				},
			}

			go func() {
				defer GinkgoRecover()
				out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
			}()

			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			goOn := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				out.Write(record, time.Unix(0, 0).UTC(), "pod.log")
				close(goOn)
			}()
			<-goOn
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				_, _ = spySink.lis.Accept()
			}()
			Consistently(done).ShouldNot(BeClosed())
		})

		It("reconnects if previous connection went away", func() {
			spySink := newTLSSpySink()
			s := syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-namespace",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}
			out := syslog.NewOut([]*syslog.Sink{&s}, nil)
			// TLS will block on waiting for handshake so the write needs
			// to occur in a separate go routine
			go func() {
				defer GinkgoRecover()
				r1 := map[interface{}]interface{}{
					"log": []byte("some-log-message"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("some-namespace"),
					},
				}
				out.Write(r1, time.Unix(0, 0).UTC(), "pod.log")
			}()
			spySink.expectReceived(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message` + "\n",
			)

			spySink.stop()
			spySink = newTLSSpySink(spySink.url())
			defer spySink.stop()

			f := func() int64 {
				r2 := map[interface{}]interface{}{
					"log": []byte("some-log-message-2"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("some-namespace"),
					},
				}
				out.Write(r2, time.Unix(0, 0).UTC(), "pod.log")
				return s.MessagesDropped()
			}
			Eventually(f).Should(BeNumerically(">=", 1))

			go func() {
				defer GinkgoRecover()
				r3 := map[interface{}]interface{}{
					"log": []byte("some-log-message-3"),
					"kubernetes": map[interface{}]interface{}{
						"namespace_name": []byte("some-namespace"),
					},
				}
				out.Write(r3, time.Unix(0, 0).UTC(), "pod.log")
			}()

			spySink.expectReceivedIncludes(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-namespace// - - [kubernetes@47450 namespace_name="some-namespace" object_name="" container_name=""] some-log-message-3` + "\n",
			)
		})

		It("writes messages via syslog-tls w/o rootCA verify", func() {
			spySink := newTLSSpySink()
			defer spySink.stop()

			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}

			out := syslog.NewOut([]*syslog.Sink{s}, nil)
			r := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-ns"),
				},
			}

			// TLS will block on waiting for handshake so the write needs
			// to occur in a separate go routine
			go func() {
				defer GinkgoRecover()
				out.Write(r, time.Unix(0, 0).UTC(), "pod.log")
			}()

			spySink.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns// - - [kubernetes@47450 namespace_name="some-ns" object_name="" container_name=""] some-log` + "\n",
			)
		})

		It("fails when connecting to non TLS endpoint", func() {
			spySink := newSpySink()
			defer spySink.stop()

			s := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
				TLS: &syslog.TLS{
					InsecureSkipVerify: true,
				},
			}

			out := syslog.NewOut(
				[]*syslog.Sink{s},
				nil,
				syslog.WithDialTimeout(time.Millisecond),
			)
			r := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-ns"),
				},
			}

			out.Write(r, time.Unix(0, 0).UTC(), "pod.log")
			Eventually(s.MessagesDropped).Should(Equal(int64(1)))
		})

		It("writes messages via syslog-tls w/ rootCA verify", func() {
			spySink := newTLSSpySink("localhost:8443")
			defer spySink.stop()

			s1 := &syslog.Sink{
				Addr:      spySink.url(),
				Namespace: "some-ns",
				TLS: &syslog.TLS{
					RootCA: "./testdata/rootCA.crt",
				},
			}
			cs1 := &syslog.Sink{
				Addr: spySink.url(),
				TLS: &syslog.TLS{
					RootCA: "./testdata/rootCA.crt",
				},
			}

			out := syslog.NewOut([]*syslog.Sink{s1}, []*syslog.Sink{cs1})
			r := map[interface{}]interface{}{
				"log": []byte("some-log"),
				"kubernetes": map[interface{}]interface{}{
					"namespace_name": []byte("some-ns"),
				},
			}

			// TLS will block on waiting for handshake so the write needs
			// to occur in a separate go routine
			go func() {
				defer GinkgoRecover()
				out.Write(r, time.Unix(0, 0).UTC(), "pod.log")
			}()

			spySink.expectReceivedOnly(
				`<14>1 1970-01-01T00:00:00+00:00 - pod.log/some-ns// - - [kubernetes@47450 namespace_name="some-ns" object_name="" container_name=""] some-log` + "\n",
			)
		})
	})
})

// This is an example of a message that is sent to the syslog output plugin.
// It is here for documentation purposes.
var _ = map[interface{}]interface{}{
	"log":    []byte("log data"),
	"stream": []byte("stdout"),
	"time":   []byte("2018-07-16T17:47:16.61514406Z"),
	"kubernetes": map[interface{}]interface{}{
		"labels": map[interface{}]interface{}{
			"component":                     []byte("kube-addon-manager"),
			"kubernetes.io/minikube-addons": []byte("addon-manager"),
			"version":                       []byte("v8.6"),
		},
		"annotations": map[interface{}]interface{}{
			"kubernetes.io/config.hash":   []byte{},
			"kubernetes.io/config.mirror": []byte{},
			"kubernetes.io/config.seen":   []byte{},
			"kubernetes.io/config.source": []byte{},
		},
		"host":           []byte("minikube"),
		"container_name": []byte("kube-addon-manager"),
		"docker_id":      []byte("some-hash"),
		"pod_name":       []byte("kube-addon-manager-minikube"),
		"namespace_name": []byte("kube-system"),
		"pod_id":         []byte("some-hash"),
	},
}
