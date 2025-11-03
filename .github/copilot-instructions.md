# Copilot Instructions for TDA596-DIT240 Distributed Systems

This repository contains course materials and lab assignments for the Distributed Systems course (TDA596/DIT240).

## Repository Structure

- `/Labs/Lab1` - Lab 1 assignments and materials
- `/Labs/Lab2` - Lab 2 assignments and materials
- `/Labs/Lab3` - Lab 3 assignments and materials
- `/Labs/Lab4` - Lab 4 assignments and materials

## Coding Guidelines

### General Principles

1. **Clarity over Cleverness**: Write code that is easy to understand and maintain, especially important for educational purposes
2. **Document Distributed Systems Concepts**: When implementing distributed systems concepts, include comments explaining the theoretical background
3. **Error Handling**: Always implement proper error handling, especially for network operations and concurrent code
4. **Concurrency Safety**: Be mindful of race conditions, deadlocks, and other concurrency issues common in distributed systems

### Distributed Systems Best Practices

1. **Network Communication**:
   - Always handle network failures gracefully
   - Implement appropriate timeouts
   - Consider message ordering and delivery guarantees
   - Document which communication protocol is being used (TCP/UDP, HTTP, RPC, etc.)

2. **Consistency and Consensus**:
   - Clearly document consistency models (strong, eventual, etc.)
   - When implementing consensus algorithms (Paxos, Raft, etc.), reference the specific algorithm variant
   - Consider edge cases like network partitions and node failures

3. **State Management**:
   - Clearly separate local state from distributed state
   - Document state synchronization mechanisms
   - Consider idempotency for operations

4. **Testing**:
   - Include unit tests for individual components
   - Consider integration tests for distributed interactions
   - Test failure scenarios (network failures, node crashes, etc.)
   - Use deterministic testing where possible

### Code Style

- Use clear and descriptive variable names
- Keep functions focused and single-purpose
- Add docstrings/comments for complex algorithms
- Follow language-specific conventions (PEP 8 for Python, etc.)

### Documentation

- Each lab should include a README explaining:
  - The objective of the lab
  - How to build and run the code
  - Any dependencies or requirements
  - Expected behavior and test cases

### Common Patterns to Follow

1. **Logging**: Use appropriate logging levels (DEBUG, INFO, WARNING, ERROR)
2. **Configuration**: Externalize configuration (ports, addresses, timeouts)
3. **Modularity**: Separate concerns (e.g., network layer, application logic, data persistence)

## What to Avoid

- Hardcoded IP addresses or ports (use configuration files or environment variables)
- Blocking operations without timeouts
- Unbounded resource usage (memory, connections, threads)
- Ignoring edge cases in distributed scenarios
- Code without proper synchronization in concurrent contexts

## When Adding New Labs

1. Create appropriate directory structure under `/Labs/LabX/`
2. Include a README.md with lab instructions
3. Add example code or starter templates if applicable
4. Include test cases or validation scripts
5. Document any external dependencies in a requirements file (requirements.txt, package.json, etc.)

## Educational Context

Remember that this is a learning environment. Code should:
- Be well-commented to aid understanding
- Demonstrate distributed systems concepts clearly
- Include references to relevant academic papers or textbooks where applicable
- Balance between production-ready practices and pedagogical clarity
