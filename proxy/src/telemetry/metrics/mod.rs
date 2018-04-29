//! Records and serves Prometheus metrics.
//!
//! # A note on label formatting
//!
//! Prometheus labels are represented as a comma-separated list of values
//! Since the Conduit proxy labels its metrics with a fixed set of labels
//! which we know in advance, we represent these labels using a number of
//! `struct`s, all of which implement `fmt::Display`. Some of the label
//! `struct`s contain other structs which represent a subset of the labels
//! which can be present on metrics in that scope. In this case, the
//! `fmt::Display` impls for those structs call the `fmt::Display` impls for
//! the structs that they own. This has the potential to complicate the
//! insertion of commas to separate label values.
//!
//! In order to ensure that commas are added correctly to separate labels,
//! we expect the `fmt::Display` implementations for label types to behave in
//! a consistent way: A label struct is *never* responsible for printing
//! leading or trailing commas before or after the label values it contains.
//! If it contains multiple labels, it *is* responsible for ensuring any
//! labels it owns are comma-separated. This way, the `fmt::Display` impl for
//! any struct that represents a subset of the labels are position-agnostic;
//! they don't need to know if there are other labels before or after them in
//! the formatted output. The owner is responsible for managing that.
//!
//! If this rule is followed consistently across all structs representing
//! labels, we can add new labels or modify the existing ones without having
//! to worry about missing commas, double commas, or trailing commas at the
//! end of the label set (all of which will make Prometheus angry).
use std::default::Default;
use std::hash::Hash;
use std::fmt::{self, Display};
use std::sync::{Arc, Mutex};
use std::time;

use indexmap::IndexMap;

use ctx;

mod counter;
mod gauge;
mod histogram;
mod labels;
mod latency;
mod record;
mod serve;

use self::counter::Counter;
use self::gauge::Gauge;
use self::histogram::Histogram;
use self::labels::{
    RequestLabels,
    ResponseLabels,
    TransportLabels,
    TransportCloseLabels
};
pub use self::labels::DstLabels;
pub use self::record::Record;
pub use self::serve::Serve;

trait FmtMetric {
    fn fmt_metric<N: Display>(&self, f: &mut fmt::Formatter, name: N) -> fmt::Result;

    fn fmt_metric_labeled<N, L>(&self, f: &mut fmt::Formatter, name: N, labels: L) -> fmt::Result
    where
        N: Display,
        L: Display;
}

#[derive(Debug, Default)]
struct Root {
    requests: RequestScopes,
    responses: ResponseScopes,
    transports: TransportScopes,
    transport_closes: TransportCloseScopes,

    start_time: Gauge,
}

#[derive(Debug)]
struct Scopes<L: Display + Hash + Eq, M> {
    scopes: IndexMap<L, M>,
}

type RequestScopes = Scopes<RequestLabels, RequestMetrics>;

#[derive(Debug, Default)]
struct RequestMetrics {
    total: Counter,
}

type ResponseScopes = Scopes<ResponseLabels, ResponseMetrics>;

#[derive(Debug, Default)]
struct ResponseMetrics {
    total: Counter,
    latency: Histogram<latency::Ms>,
}

type TransportScopes = Scopes<TransportLabels, TransportMetrics>;

#[derive(Debug, Default)]
struct TransportMetrics {
    open_total: Counter,
    open_connections: Gauge,
    write_bytes_total: Counter,
    read_bytes_total: Counter,
}

type TransportCloseScopes = Scopes<TransportCloseLabels, TransportCloseMetrics>;

#[derive(Debug, Default)]
struct TransportCloseMetrics {
    close_total: Counter,
    connection_duration: Histogram<latency::Ms>,
}

/// Construct the Prometheus metrics.
///
/// Returns the `Record` and `Serve` sides. The `Serve` side
/// is a Hyper service which can be used to create the server for the
/// scrape endpoint, while the `Record` side can receive updates to the
/// metrics by calling `record_event`.
pub fn new(process: &Arc<ctx::Process>) -> (Record, Serve){
    let metrics = Arc::new(Mutex::new(Root::new(process)));
    (Record::new(&metrics), Serve::new(&metrics))
}

// ===== impl Root =====

impl Root {
    pub fn new(process: &Arc<ctx::Process>) -> Self {
        let t0 = process.start_time
            .duration_since(time::UNIX_EPOCH)
            .expect("process start time")
            .as_secs();

        Self {
            start_time: t0.into(),
            .. Root::default()
        }
    }

    fn request(&mut self, labels: RequestLabels) -> &mut RequestMetrics {
        self.requests.scopes.entry(labels)
            .or_insert_with(RequestMetrics::default)
    }

    fn response(&mut self, labels: ResponseLabels) -> &mut ResponseMetrics {
        self.responses.scopes.entry(labels)
            .or_insert_with(ResponseMetrics::default)
    }

    fn transport(&mut self, labels: TransportLabels) -> &mut TransportMetrics {
        self.transports.scopes.entry(labels)
            .or_insert_with(TransportMetrics::default)
    }

    fn transport_close(&mut self, labels: TransportCloseLabels) -> &mut TransportCloseMetrics {
        self.transport_closes.scopes.entry(labels)
            .or_insert_with(TransportCloseMetrics::default)
    }
}

fn fmt_help(f: &mut fmt::Formatter, name: &str, kind: &str, help: &str) -> fmt::Result {
    writeln!(f, "# HELP {} {}", name, help)?;
    writeln!(f, "# TYPE {} {}", name, kind)?;

    Ok(())
}

impl fmt::Display for Root {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        self.requests.fmt(f)?;
        self.responses.fmt(f)?;
        self.transports.fmt(f)?;
        self.transport_closes.fmt(f)?;

        fmt_help(f, "process_start_time_seconds", "gauge",
            "The time this process started in seconds since the UNIX epoch")?;
        self.start_time.fmt_metric(f, "process_start_time_seconds")?;

        Ok(())
    }
}

// ===== impl Scopes =====

impl<L: Display + Hash + Eq, M> Default for Scopes<L, M> {
    fn default() -> Self {
        Scopes { scopes: IndexMap::default(), }
    }
}

// ===== impl RequestScopes =====

impl fmt::Display for RequestScopes {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        if self.scopes.is_empty() {
            return Ok(());
        }

        fmt_help(f, "request_total", "counter", "Total count of routed HTTP requests.")?;
        for (labels, scope) in &self.scopes {
            scope.total.fmt_metric_labeled(f, "request_total", labels)?;
        }

        Ok(())
    }
}

// ===== impl ResponseScopes =====

impl fmt::Display for ResponseScopes {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        if self.scopes.is_empty() {
            return Ok(());
        }

        fmt_help(f, "response_total", "counter", "Total count of HTTP responses")?;
        for (labels, scope) in &self.scopes {
            scope.total.fmt_metric_labeled(f, "response_total", labels)?;
        }

        fmt_help(f, "response_latency_ms", "histogram",
            "Elapsed times between a request's headers being received \
            and its response stream completing")?;
        for (labels, scope) in &self.scopes {
            scope.latency.fmt_metric_labeled(f, "response_latency_ms", labels)?;
        }

        Ok(())
    }
}

// ===== impl TransportScopes =====

impl fmt::Display for TransportScopes {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        if self.scopes.is_empty() {
            return Ok(());
        }

        fmt_help(f, "tcp_open_total", "counter", "Total count of opened connections")?;
        for (labels, scope) in &self.scopes {
            scope.open_total.fmt_metric_labeled(f, "tcp_open_total", labels)?;
        }

        fmt_help(f, "tcp_open_connections", "gauge", "Number of currently-open connections")?;
        for (labels, scope) in &self.scopes {
            scope.open_connections.fmt_metric_labeled(f, "tcp_open_connections", labels)?;
        }

        fmt_help(f, "tcp_read_bytes_total", "counter", "Total count of bytes read")?;
        for (labels, scope) in &self.scopes {
            scope.read_bytes_total.fmt_metric_labeled(f, "tcp_read_bytes_total", labels)?;
        }

        fmt_help(f, "tcp_write_bytes_total", "counter", "Total count of bytes written")?;
        for (labels, scope) in &self.scopes {
            scope.write_bytes_total.fmt_metric_labeled(f, "tcp_write_bytes_total", labels)?;
        }

        Ok(())
    }
}

// ===== impl TransportCloseScopes =====

impl fmt::Display for TransportCloseScopes {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        if self.scopes.is_empty() {
            return Ok(());
        }

        fmt_help(f, "tcp_close_total", "counter", "Total count of closed connections")?;
        for (labels, scope) in &self.scopes {
            scope.close_total.fmt_metric_labeled(f, "tcp_close_total", labels)?;
        }

        fmt_help(f, "tcp_connection_duration_ms", "histogram", "Connection lifetimes")?;
        for (labels, scope) in &self.scopes {
            scope.connection_duration.fmt_metric_labeled(f, "tcp_connection_duration_ms", labels)?;
        }

        Ok(())
    }
}
