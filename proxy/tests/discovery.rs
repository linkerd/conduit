mod support;
use self::support::*;

#[test]
fn outbound_asks_controller_api() {
    let _ = env_logger::init();

    let srv = server::new().route("/", "hello").route("/bye", "bye").run();
    let ctrl = controller::new()
        .destination("test.conduit.local", srv.addr)
        .run();
    let proxy = proxy::new().controller(ctrl).outbound(srv).run();
    let client = client::new(proxy.outbound, "test.conduit.local");

    assert_eq!(client.get("/"), "hello");
    assert_eq!(client.get("/bye"), "bye");
}

#[test]
fn outbound_reconnects_if_controller_stream_ends() {
    let _ = env_logger::init();

    let srv = server::new().route("/recon", "nect").run();
    let ctrl = controller::new()
        .destination_close("test.conduit.local")
        .destination("test.conduit.local", srv.addr)
        .run();
    let proxy = proxy::new().controller(ctrl).outbound(srv).run();
    let client = client::new(proxy.outbound, "test.conduit.local");

    assert_eq!(client.get("/recon"), "nect");
}

#[test]
fn outbound_updates_newer_services() {
    let _ = env_logger::init();

    //TODO: when the support server can listen on both http1 and http2
    //at the same time, do that here
    let srv = server::http1().route("/h1", "hello h1").run();
    let ctrl = controller::new()
        .destination("test.conduit.local", srv.addr)
        .run();
    let proxy = proxy::new().controller(ctrl).outbound(srv).run();
    // the HTTP2 service starts watching first, receiving an addr
    // from the controller
    let client1 = client::http2(proxy.outbound, "test.conduit.local");
    client1.get("/h2"); // 500, ignore

    // a new HTTP1 service needs to be build now, while the HTTP2
    // service already exists, so make sure previously sent addrs
    // get into the newer service
    let client2 = client::http1(proxy.outbound, "test.conduit.local");
    assert_eq!(client2.get("/h1"), "hello h1");
}

#[test]
#[ignore]
fn outbound_times_out() {
    // Currently, the outbound router will wait forever until discovery tells
    // it where to send the request. It should probably time out eventually.
}
