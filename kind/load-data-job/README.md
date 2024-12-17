# load-data-job

Due to database schema changes cross major/minor NetBox versions, we have to `patch` the SQL files and demo data link on-the-fly.

The default values stems from the NetBox 4.1.x version. So the patching will only happen for 3.7.x and 4.0.x versions. 

Please see `../local-env.sh`, that's where all the patching happen.
