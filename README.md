# go-auto-instrument

> Automatically add Trace Spans to Go methods and functions

* Code is not modified
* No new dependencies

TODO:
- generate new class wrapper in `/<pkg>/trace` of original one that adds spans
- generate functions with same name in `/<pkg>/trace` of original one that adds spans
- ask to switch pkg in original code

## Motivation

It is laborious to add tracing code to every function manually.
The code repeats 99% of time.
Other languages can either modify code or have wrapper notations that makes even manual tracing much less laborious.

As of `2022-11-06`, official Go does not support automatic function traces. https://go.dev/doc/diagnostics
> Is there a way to automatically intercept each function call and create traces?  
> Go doesn’t provide a way to automatically intercept every function call and create trace spans. You need to manually instrument your code to create, end, and annotate spans.

Thus, providing automated version to add Trace Spans annotation.

## Appendix A: Related Work

* https://github.com/hedhyw/otelinji — It modifies code to inline tracing fuinction calls. It is flexible and handles Go code well (name of `context.Context`, `error`, comments). Meanwhile, current project focuses on minimal code, minimal dependencies, and no changes to original code.
* https://github.com/open-telemetry/opentelemetry-go-instrumentation - (In-Development) official eBPF based Go auto instrumentation
* https://github.com/keyval-dev/opentelemetry-go-instrumentation - eBPF based Go auto instrumentation of _pre-selected_ libraries

## Appendix B: Other Languages

### Java

Java runtime modifies bytecode of methods on load time that adds instrumentation calls.
Pre-defined libraries are instrumented (http, mysql, etc).

✅ Very short single line decorator statement can be used to trace selected methods.

Datadog
```java
import datadog.trace.api.Trace

public class BackupLedger {
  @Trace
  public void write(List<Transaction> transactions) {
    for (Transaction transaction : transactions) {
      ledger.put(transaction.getId(), transaction);
    }
  }
}
```

OpenTelemetry
```java
import io.opentelemetry.instrumentation.annotations.WithSpan;

public class MyClass {
  @WithSpan
  public void myMethod() {
      <...>
  }
}
```

✅ Automatic instrumentation of all functions is also possible.

Datadog supports wildcard for list of methods to trace.

> dd.trace.methods  
> Environment Variable: DD_TRACE_METHODS  
> Default: null  
> Example: package.ClassName[method1,method2,...];AnonymousClass$1[call];package.ClassName[*]  
> List of class/interface and methods to trace. Similar to adding @Trace, but without changing code. Note: The wildcard method support ([*]) does not accommodate constructors, getters, setters, synthetic, toString, equals, hashcode, or finalizer method calls

```bash
java -javaagent:/path/to/dd-java-agent.jar -Ddd.service=web-app -Ddd.env=dev -Ddd.trace.methods="*" -jar path/to/application.jar
```

* [Java Auto-Instrumentation](https://docs.oracle.com/javase/8/docs/api/java/lang/instrument/package-summary.html)
* [Datadog Java Auto-Instrumentation](https://docs.datadoghq.com/tracing/trace_collection/dd_libraries/java/?tab=containers#automatic-instrumentation)
* [Datadog Java Tracing Config](https://docs.datadoghq.com/tracing/trace_collection/library_config/java/#ddtracemethods)
* [Datadog Instrumentation Business Logic](https://docs.datadoghq.com/tracing/guide/instrument_custom_method/?code-lang=java)
* [Javaassist](https://www.javassist.org)

### Python

Python monkeypatching of functions at runtime is used to add instrumentation calls.
Pre-defined libraries are instrumented (http, mysql, etc).

✅ Very short single line decorator statement can be used to trace selected methods.

Datadog
```python
from ddtrace import tracer

class BackupLedger:
    @tracer.wrap()
    def write(self, transactions):
        for transaction in transactions:
            self.ledger[transaction.id] = transaction
```

OpenTelemetry
```python
@tracer.start_as_current_span("do_work")
def do_work():
    print("doing some work...")
```

⚠️ Automatic instrumentation of all functions is also possible via monkeypatching (fidning stable library is pending).

* [OpenTelemetry Python Instrumentation](https://opentelemetry.io/docs/instrumentation/python/automatic/#overview)
* [Blog: Timescale: OpenTelemetry and Python: A Complete Instrumentation Guide](https://www.timescale.com/blog/opentelemetry-and-python-a-complete-instrumentation-guide/)
* https://github.com/harshitandro/Python-Instrumentation

### C++

❌ Only manual instrumentation.
