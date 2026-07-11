# Changelog

## 0.1.0

- Initial release.
- Injection grammar highlighting `@Annotation(args)` inside Go `//` and `/* */`
  comments: names, argument keys, `=`, strings/raw strings, numbers, booleans,
  `null`, arrays/objects, and dotted enums.
- Snippets for the common goboot annotations (`@Service`, `@RestController`,
  `@GetMapping`/verbs, `@Repository`, `@Query`/`@Exec`, `@Transactional`,
  `@Scheduled`, `@ConfigurationProperties`, `@ControllerAdvice`, …).
