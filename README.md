

## Assumption:
1. amount less than 1 paisa is rounded down.
Explanation:
If A pays rs 100 for a group of 3, split equally.
B & C have to pay Rs 33.33 each. (A has to bear Rs 33.34)
This would be notisable for a very large group, difference is added to first user for simplicity. 

2. Application recives more read traffic than write traffic.

3. Overall outstanding balance = amout that has to be received - amount that needs to be payed.

## Testing:
Postman collection is added in Resources folder. 


## DB Schema
[Database Schema](db/schema.md)


## Steps to run
