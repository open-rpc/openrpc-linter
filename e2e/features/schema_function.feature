Feature: JSON Schema rule function
  Rule authors can use functionOptions as a JSON Schema for the selected value.

  Scenario: Empty methods fails the recommended methods length rule
    Given a rules file with the methods length schema rule
    And an OpenRPC document with no methods
    When I run the linter
    Then the lint should fail
    And the lint output should mention "Value does not match schema"

  Scenario: Non-empty methods passes the recommended methods length rule
    Given a rules file with the methods length schema rule
    And an OpenRPC document with one method
    When I run the linter
    Then the lint should pass
    And the lint output should mention "All 1 rules passed"
