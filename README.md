# go-otel-auto-instrument

> Automatically add Trace Spans to Go functions

- generate new class wrapper in `/<pkg>/trace` of original one that adds spans + need to ask original code to wrap classes

## Performance Impact

Since we are adding extra calls in all functions, compiler may change its behavior, this can lead to performance deterioration.

### Function Inlining

????

## Appendix A: Related Work

* https://github.com/hedhyw/otelinji — It modifies code to inline tracing fuinction calls. It is flexible and handles Go code well (name of `context.Context`, `error`, comments). Meanwhile, current project focuses on minimal code and dependencies.

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

### Appendix C: Paths Not Taken

- modify existing file by inlining trace calls to top of function — to avoid modifying original user code as much as possible
- generate files within same packages - to avoid corrupting code coverage and code statistics
