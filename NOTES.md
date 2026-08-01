# Decisions

# Flat terraform structure

Given that I don't have complex conditional logic, nor big reusable custom resources, I've opted out for a flat structure for all terraform resources. I've split them in files so that it's easier to mentally group and read.

Using a flat structure has the benefits of:

- less code and it's less error prone, since variables and outputs take a lot of space and are places where it's easy to missconfig.
- easier to read, since there are no custom modules with my own defined variables. By using standard terraform modules like `google`, everyone can read the resources and the code is familiar.

# Current design is a bit excessive

A custom load balancer and public IP is really not needed for this simple app, since we could simply use Google's Front End (GFE). But given that the requirements I've opted for using the load balancer with public static IP.
