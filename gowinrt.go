/**
 * File: gowinrt.go
 * Project: gowinrt
 * Created Date: 2026‑03‑29T01:28:58.5858+01:00
 * Author: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Last Modified: 2026‑03‑29T01:32:19.1919+01:00
 * Modified By: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Copyright © 2026 ZanderCodes (Julian Zander). All rights reserved.
 */

package gowinrt

// common
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.IClosable
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.IAsyncOperation`1
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.AsyncOperationCompletedHandler`1
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.AsyncStatus
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.IAsyncInfo --method-filter !*
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.IAsyncOperationWithProgress`2
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.AsyncOperationProgressHandler`2
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.AsyncOperationWithProgressCompletedHandler`2
////go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.DateTime
////go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.Deferral
////go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.DeferralCompletedHandler
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.IReference`1

// event
////go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.TypedEventHandler`2
////go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.EventRegistrationToken

// vector
////go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.Collections.IVector`1
//go:generate go run ./cmd/gowinrt -v --class Windows.Foundation.Collections.IVectorView`1

// buffer
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IBuffer
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.Buffer --method-filter !CreateCopyFromMemoryBuffer --method-filter !CreateMemoryBufferOverIBuffer
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IDataReader --method-filter ReadBytes --method-filter !*
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.DataReader --method-filter FromBuffer --method-filter ReadBytes --method-filter !*
////go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IDataWriter --method-filter WriteBytes --method-filter DetachBuffer --method-filter !*
////go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.DataWriter --method-filter WriteBytes --method-filter DetachBuffer --method-filter DataWriter --method-filter Close --method-filter !*

// stream
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IRandomAccessStreamReference
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.RandomAccessStreamReference --method-filter OpenReadAsync --method-filter !*
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IRandomAccessStream
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IInputStream
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IOutputStream
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.InputStreamOptions
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IContentTypeProvider
//go:generate go run ./cmd/gowinrt -v --class Windows.Storage.Streams.IRandomAccessStreamWithContentType --inheritance --method-filter !Close --method-filter !WriteAsync --method-filter !FlushAsync

// media
//go:generate go run ./cmd/gowinrt -v --class Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager --method-filter RequestAsync --method-filter GetCurrentSession --method-filter !*
//go:generate go run ./cmd/gowinrt -v --class Windows.Media.Control.GlobalSystemMediaTransportControlsSession --method-filter TryGetMediaPropertiesAsync --method-filter !*
//go:generate go run ./cmd/gowinrt -v --class Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties
