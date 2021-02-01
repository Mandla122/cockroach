import React from "react";
import * as protos from "@cockroachlabs/crdb-protobuf-client";
import { stdDevLong } from "src/util/appStats";
import { Duration, Bytes } from "src/util/format";
import classNames from "classnames/bind";
import styles from "./barCharts.module.scss";
import { bar, formatTwoPlaces, longToInt, approximify } from "./utils";
import { barChartFactory } from "./barChartFactory";

type StatementStatistics = protos.cockroach.server.serverpb.StatementsResponse.ICollectedStatementStatistics;
const cx = classNames.bind(styles);

const countBars = [
  bar("count-first-try", (d: StatementStatistics) =>
    longToInt(d.stats.first_attempt_count),
  ),
];

const rowsReadBars = [
  bar("rows-read", (d: StatementStatistics) => d.stats.rows_read.mean),
];

const bytesReadBars = [
  bar("bytes-read", (d: StatementStatistics) => d.stats.bytes_read.mean),
];

const latencyBars = [
  bar("bar-chart__parse", (d: StatementStatistics) => d.stats.parse_lat.mean),
  bar("bar-chart__plan", (d: StatementStatistics) => d.stats.plan_lat.mean),
  bar("bar-chart__run", (d: StatementStatistics) => d.stats.run_lat.mean),
  bar(
    "bar-chart__overhead",
    (d: StatementStatistics) => d.stats.overhead_lat.mean,
  ),
];

const maxMemUsageBars = [
  bar("max-mem-usage", (d: StatementStatistics) => d.stats.max_mem_usage?.mean),
];

const networkBytesBars = [
  bar("network-bytes", (d: StatementStatistics) => d.stats.bytes_sent_over_network?.mean),
]

const retryBars = [
  bar(
    "count-retry",
    (d: StatementStatistics) =>
      longToInt(d.stats.count) - longToInt(d.stats.first_attempt_count),
  ),
];

const rowsReadStdDev = bar(cx("rows-read-dev"), (d: StatementStatistics) =>
  stdDevLong(d.stats.rows_read, d.stats.exec_stat_collection_count),
);
const bytesReadStdDev = bar(cx("rows-read-dev"), (d: StatementStatistics) =>
  stdDevLong(d.stats.bytes_read, d.stats.exec_stat_collection_count),
);
const latencyStdDev = bar(
  cx("bar-chart__overall-dev"),
  (d: StatementStatistics) => stdDevLong(d.stats.service_lat, d.stats.count),
);
const maxMemUsageStdDev = bar(
  cx("max-mem-usage-dev"),
  (d: StatementStatistics) =>
    stdDevLong(d.stats.max_mem_usage, d.stats.exec_stat_collection_count),
);
const networkBytesStdDev = bar(
  cx("network-bytes-dev"),
  (d: StatementStatistics) =>
    stdDevLong(d.stats.bytes_sent_over_network, d.stats.exec_stat_collection_count),
);

export const countBarChart = barChartFactory("grey", countBars, approximify);
export const rowsReadBarChart = barChartFactory(
  "grey",
  rowsReadBars,
  approximify,
  rowsReadStdDev,
  formatTwoPlaces,
);
export const bytesReadBarChart = barChartFactory(
  "grey",
  bytesReadBars,
  Bytes,
  bytesReadStdDev,
);
export const latencyBarChart = barChartFactory(
  "grey",
  latencyBars,
  v => Duration(v * 1e9),
  latencyStdDev,
);
export const maxMemUsageBarChart = barChartFactory(
    "grey",
    maxMemUsageBars,
    Bytes,
    maxMemUsageStdDev,
)
export const networkBytesBarChart = barChartFactory(
  "grey",
  networkBytesBars,
  Bytes,
  networkBytesStdDev,
)

export const retryBarChart = barChartFactory("red", retryBars, approximify);
