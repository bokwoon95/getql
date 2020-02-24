```
GET localhost:8080?
    SEL=a&SEL=b&SEL=c
    &FRM=some_table
    &COL.1=fruit
    &OPR.1=EQ
    &VAL.1=apple
    &COL.2=rank
    &OPR.2=BETWEEN
    &VAL.2=9&VAL.2=10
    &COL.3=color
    &OPR.3=IN
    &VAL.3=red&VAL.3=green&VAL.3=blue
    &AOR.4=OR
    &COL.4.1=user
    &OPR.4.1=EQ
    &VAL.4.1=john
    &COL.4.2=admin
    &OPR.4.2=EQ
    &VAL.4.2=john
    &COL.4.3=name
    &OPR.4.3=NE
    &VAL.4.3=sammy
    &ORD.1=id
    &ORD.2=date&ORD.2=ASC
    &ORD.3=DESC&ORD.3=time

Results In =>
    SELECT
        a, b, c
    FROM
        some_table
    WHERE
        fruit = apple
        AND rank BETWEEN 9 AND 10
        AND color IN ('red', 'green', 'blue')
        AND (
            user = 'john' 
            OR admin = 'john'
            OR name <> 'sammy'
        )
    ORDER BY
        id ASC
        ,date ASC
        ,time DESC
    ;
```
