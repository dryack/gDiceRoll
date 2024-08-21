# gDiceRoll

## Functionality
* Provide a DSL for the rolling of dice
* Provide a comprehensive framework for the testing of dice rolling expressions
* Provide a caching layer for dice rolls that have been seen before, so that complex operations to determine the statistical values of the roll don't need to be repeated
* Output JSON results that contain:
  * The dice roll as parsed 
  * The final value of the dice roll
  * The average value of the dice roll
  * The 0th, 5th, 10th, 25th, 75th, 90th, 95th, and 100th percentile possible values of the dice roll
  * The skewness of the dice roll expression
  * Information necessary to plot the distribution of the dice roll on a graph or bar chart
  * Expressions or sub-expressions that result in divide-by-zero will ALWAYS result in a divide-by-one instead, but will there will be a divided-by-zero boolean set to true in the JSON
* JSON output must be comprehensive enough to support all client needs
* Clients will expect a RESTful API
* In the future clients should instead be able to utilize gRPC
* Provide a web admin interface that can, among other things, display realtime dice roll requests, logs, metrics, and statistics relating to the use of the server
### DSL
* Utilize PEMDAS to resolve the order of operations in a dice roll expression
* Allow flags on a dice roll expression, which are signified by a `#` followed by two or three alphabetic characters (ie. #nz or #gnz)
  * Multiple flags are seperated by commas
  * Flags always terminate an expression or sub-expression (ie. 2d6-2#nz+2d6 would have a minimum value of 3)
  * Global flags apply to each sub-expression (ie. (2d6-2)+2d6-2#gnz would have a minimum value of 2)
    * In this case, we'd evaluate `2d6-2` twice, each time with `1` as its minimum value, and then add them together
  * Sub-expressions are evaluated independently, from left to right
  * Parentheses to explicitly define sub-expressions
* The use of `k` to signify keeping the highest rolls of n dice (ie. 4d6k3 would mean keeping the three highest rolls out of 4)
* Useless operations will be automatically deleted (ie. +/-0 has no meaning)
* Definitions that include a divide-by-zero will return an error message, not a result
* Definitions that include a 0 divided by an expression or sub-expression will return an error message, not a result
* The use of `!` to signify exploding dice.
  * `3d6!` - would mean roll 3 six-sided dice, and for each maximum roll (6), roll an additional die.
  * `3d6!5` - would mean roll 3 six-sided dice, and for each roll (5 or 6), roll an additional die.
  * `3d6!!` - would mean each additional roll of (6) would also roll
  * `3d6!!5` -  would mean roll 3 six-sided dice, and for each roll (5 or 6), roll an additional die, each of which would also explode on a (5 or 6)
### Statistical Operations
* Dice rolls are simulated using the Monte Carlo method, in order to determine any statistical results that can't be computed directly
  * We will generally simulate a roll 1 million or 2 million times 
  * The Monte Carlo simulation utilizes go routines up to the number of CPU cores available on the system minus 4
  * The result of each simulation is returned on a channel
  * The collected results are sorted and returned
* The average value of the dice roll
* The 0th, 5th, 10th, 25th, 75th, 90th, 95th, and 100th percentile possible values of the dice roll
* The skewness of the dice roll expression
* The variance of the dice roll expression
* The entropy of the dice roll expression
* The Kurtosis of the dice roll expression
* The Standard Deviation of the dice roll expression
* The Variance-to-Mean Ratio of the dice roll expression
### Similar Roll Suggestions
* Maintain a cache of previously rolled expressions and their statistics.
* When a user rolls dice, suggest similar alternative rolls based on the following criteria:
  * Primary factor: Similar range (minimum and maximum possible values).
  * Secondary factor: Different kurtosis (shape of the probability distribution).
  * Provide 3-5 suggestions for each roll, prioritizing expressions with the most similar range but different probability curves.
* Balancing Range Similarity and Kurtosis
  * Use a weighted scoring system. For example: score = (range_difference * 0.7) + (kurtosis_difference * 0.3)
  * Categorize rolls into "buckets" based on range, then sort within each bucket by kurtosis difference
  * Where possible, ensure diversity in suggestions by including at least one option with similar range but higher kurtosis, one with lower kurtosis, and one with a slightly different range but very different kurtosis
### Explanation of Suggestions
* For each suggested roll, explain why it's similar and how it differs from the original roll.
* Focus on explaining the differences in probability distribution, using terms like:
* "More peaked distribution" (higher kurtosis)
* "Flatter distribution" (lower kurtosis)
* "More likely to give extreme results"
* "More likely to give results closer to the middle of the range"
### Example Interaction
```text
Roll result: 11 (individual rolls: 3, 4, 4)

Statistics: Min 3, Max 18, Average 10.5, Variance 5.25, Kurtosis -0.37

Similar roll suggestions:
* "1d20-1" (Range: 0-19)
Explanation: This alternative has a flatter distribution. It's equally likely to roll any number in the range.

* "2d10-1" (Range: 1-19)
Explanation: This alternative has a slightly flatter distribution than 3d6. It's less likely to give results at the extreme ends of the range.

* "4d4+2" (Range: 6-18)
Explanation: This alternative has a more peaked distribution. It's more likely to give results closer to the middle of the range.
```

### Technical Considerations
* Implement efficient caching and retrieval of previous roll statistics.
* Ensure the similarity comparison algorithm is performant, especially as the cache of rolls grows
* Consider using a database or persistent storage for the roll cache to maintain suggestions across sessions
* Divide-by-Zero Handling:
  * For clarity in roll breakdowns and statistical calculations:
    * Include a specific field in the JSON output to indicate where divide-by-one substitutions occurred
    * In roll breakdowns, use a notation like `3 ÷ (1*)` where the asterisk indicates a divide-by-zero substitution
    * Add a warning or note in the output explaining the impact on statistical calculations
* Error handling:
  * Define a sensible standard error format for the JSON output (and consider how we'd handle the same for gRPC)
* Caching:
  * Cache sub-expressions independently
  * An in-memory cache will provide rapid lookups
  * An appropriate database will provide the complete history of lookups
  * We'll be using DragonflyDB (https://www.dragonflydb.io) which is mostly API compatible with Redis, and can use the `go-redis` package
  * Use connection pooling
  * Caching Stategy:
    1. When a dice roll expression is evaluated:
       * Check Dragonfly for the results (other than the dice roll result itself, which is random)
       * If found, obtain the random result, and return it together with the cached results
       * If not found, check the bloom filter in Dragonfly to see if we expect to find the results in Postgres
       * If we get a positive return from the bloom filter, check Postgres
       * If not found in Postgres - or the bloom filter answered negative to our query, compute the result
       * Store the computed result in Dragonfly with an appropriate TTL
       * Asynchronously store results in PostgresSQL as a JSON blob
    2. Periodic Syncing
       * Sync data from Dragonfly to Postgres
       * This will be a background job
    3. Ensure we've got a mechanism to manually invalidate cache entries when necessary (if we change the algorithms we use to compute statistics for example)
    4. In the event of an inconsistency between database and cache, we will recompute the result and store in both locations as normal
  * Implement logging/monitoring of cache hits/misses, etc.
* Database:
  * PostgresSQL (probably using JSON blobs for results storage)
  * Use connection pooling
* Ensure the statistical system is easily extensible; this must include the Monte Carlo simulation as well
* Documentation:
  * A complete guide to using the DSL
  * API docs
  * Explanations of statistical concepts involved
  * OpenAPI for REST API
  * protobuf for gRPC
  * GitHub markdown for everything else
* Utilize a timeout context to prevent infinite loops or expressions that take too long
  * for example:  `1d2!!` could theoretically roll forever
  * return a detailed error explaining why the expression timed out so the user can avoid it in the future
  * timeout will be configured by the server, but discoverable by the client
* Implement API versioning to prevent the breaking of existing clients as changes are made
  * RESTful API versioning will be done using Schema Versioning - we don't expect many substantial changes, so the added complexity while evolving will be manageable
* Security:
  * Establish appropriate rate limits for requests to the server 1 request per 5 seconds per client should be enough, but this should be configurable
  * Input sanitization to protect the cache and database, etc
* Tests
  * should include complete unit tests and performance testing
  * tests that can ensure the queueing system is working
* Performance metrics for the system should be tracked over time
* System should provide a simple queue for requests, so that multiple clients can utilize the server at once
* We expect to deploy using Docker Compose, in order to have a database, cache, and the system easily deployable
* Use Koanf for configuration
* Use Gin for serving the REST api
* Use Zerolog for logging
* Use github.com/alecthomas/participle/v2 for DSL definition
* Admin Website:
  * Utilize Argon2 and JWT for logging in, with a Postgres user table backing the process