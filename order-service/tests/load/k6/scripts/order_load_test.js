import http from "k6/http";
import { check } from "k6";
import { Rate, Trend, Counter } from "k6/metrics";

const errorRate = new Rate("errors");
const orderCreationTime = new Trend("order_creation_time");
const successfulOrders = new Counter("successful_orders");
const failedOrders = new Counter("failed_orders");

export const options = {
    scenarios: {
        constant_load: {
            executor: "constant-arrival-rate",
            rate: 1000,
            timeUnit: "1s",
            duration: "30s",
            preAllocatedVUs: 200,
            maxVUs: 1000,
        },
    },
    thresholds: {
        http_req_duration: ["p(95)<500"],
        http_req_failed: ["rate<0.1"],
        order_creation_time: ["p(95)<1000"],
    },
};

const BASE_URL = "http://localhost:8080";

const testProducts = [
    { id: 1, price: 100000 },
    { id: 2, price: 100000 },
];

export default function () {
    const product =
        testProducts[Math.floor(Math.random() * testProducts.length)];

    const payload = JSON.stringify({
        productId: product.id,
        totalPrice: product.price,
    });

    const params = {
        headers: {
            "Content-Type": "application/json",
        },
        timeout: "10s",
    };

    const startTime = Date.now();

    const response = http.post(`${BASE_URL}/orders`, payload, params);

    const endTime = Date.now();
    const duration = endTime - startTime;
    orderCreationTime.add(duration);

    const success = check(response, {
        "status is 200 or 201": (r) => r.status === 200 || r.status === 201,
        "response has order data": (r) => {
            try {
                const body = JSON.parse(r.body);
                return body && (body.id || body.ID);
            } catch (e) {
                return false;
            }
        },
        "response time < 2s": (r) => r.timings.duration < 2000,
    });

    if (success) {
        successfulOrders.add(1);
    } else {
        failedOrders.add(1);
        errorRate.add(1);
        console.log(
            `Request failed: Status ${response.status}, Body: ${response.body}`
        );
    }
}

export function setup() {
    console.log("Starting load test for Order Service");
    console.log("Target: 1000 orders/second for 30 seconds");
    console.log("Total expected orders: ~30,000");

    const warmupPayload = JSON.stringify({
        productId: 1,
        totalPrice: 1000,
    });

    const warmupParams = {
        headers: { "Content-Type": "application/json" },
    };

    for (let i = 0; i < 5; i++) {
        http.post(`${BASE_URL}/orders`, warmupPayload, warmupParams);
    }

    console.log("Warmup completed");
}

export function teardown(data) {
    console.log("Load test completed");
    console.log("Check the metrics above for detailed results");
}
