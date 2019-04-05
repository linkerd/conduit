const assert = require('assert');
    describe('table check', function() {
    it('should check the page for proper table structure', () => {
        browser.url('http://localhost:7777/replicationcontrollers');
        browser.waitUntil(() => {return $('table').isExisting();
          }, 10000, 'expected table to exist');
        http_headers=["Namespace", "Replication Controller","Meshed", "Success Rate", "RPS", "P50 Latency", 
        "P95 Latency", "P99 Latency", "TLS", "Grafana"];
        tcp_headers=["Namespace","Replication Controller","Meshed","Connections", "Read Bytes / sec","Write Bytes / sec", "Grafana"];
        const http=$$("table")[0].$("thead").$$("th").map(item=>item.getText());
        console.log("http headers:"+http);
        assert(http.join('')==http_headers.join(''));
        const tcp=$$("table")[1].$("thead").$$("th").map(item=>item.getText());
        console.log("tcp headers:"+tcp);
        assert(tcp.join('')==tcp_headers.join(''));
    });
});