# Component

A **Component** serves as an essential building block within a **Pipeline**.

## Component Types
Components are organized under the following
categories:

- Generic
- AI
- Data
- Application
- Operator

### Generic

**Generic** components serve as the foundational elements that support other
components within a pipeline to execute complex or combined tasks. For instance,
an **Iterator** component processes each element of an array by applying an
operation determined by a collection of nested components.

### AI

**AI** components play a crucial role in transforming unstructured data into
formats that are easy to interpret and analyze, thereby facilitating the
extraction of valuable insights. These components integrate with AI models from
various providers, whether it's the primary Instill Model or those from
third-party AI vendors. They are defined and initialized in the [`ai`](../ai)
package.

### Data

**Data** components play a crucial role in establishing connections with remote
data sources, such as IoT devices (e.g., IP cameras), cloud storage services
(e.g., GCP Cloud Storage, AWS S3), data warehouses, or vector databases (e.g.,
Pinecone). These components act as the bridge between pipeline and various external
data sources. Their primary function is to enable seamless data exchange,
enhancing pipeline's capability to work with diverse data sources
effectively. They are defined and initialized in the [`data`](../data) package.

### Application

**Application** components are used to seamlessly integrate various 3rd-party
application services. They are defined and initialized in the
[`application`](../application) package.

### Operator

**Operator** components perform data transformations inside the pipeline. They
are defined and initialized in the [`operator`](../operator) package.

## Contributing

Please refer to the [Contributing Guidelines](../../.github/CONTRIBUTING.md) for
more details.

## License

See the [LICENSE](./LICENSE) file for licensing information.
