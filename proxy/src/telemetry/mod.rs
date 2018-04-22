use std::sync::Arc;

use futures_mpsc_lossy;

use ctx;

mod control;
pub mod event;
pub mod metrics;
pub mod sensor;
pub mod tap;

pub use self::control::{Control, MakeControl};
pub use self::event::Event;
pub use self::sensor::Sensors;

/// Creates proxy-specific runtime telemetry.
///
/// [`Sensors`] hide the details of how telemetry is recorded, but expose proxy utilties
/// that support telemetry.
///
/// [`Control`] drives processing of all telemetry events for tapping as well as metrics
/// aggregation.
///
/// # Arguments
/// - `capacity`: the size of the event queue.
///
/// [`Sensors`]: struct.Sensors.html
/// [`Control`]: struct.Control.html
pub fn new(
    process: &Arc<ctx::Process>,
    capacity: usize,
) -> (Sensors, MakeControl) {
    let (tx, rx) = futures_mpsc_lossy::channel(capacity);
    let s = Sensors::new(tx);
    let c = MakeControl::new(rx, process);
    (s, c)
}
