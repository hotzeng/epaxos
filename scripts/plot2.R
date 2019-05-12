library(tidyr);
library(dplyr);
library(ggplot2);

data <- read.csv(file='report/plot2.csv', header=FALSE, sep=',');
colnames(data) <- c('servers', 'records', 'pipe', 'collision', 'inter', 'intra', 'jitter', 'id', 'throughput');
data$id <- seq.int(nrow(data));
data <- data %>%
	mutate(collision=collision / 100, jitter=jitter / 100) %>%
	select(-servers, -records, -pipe, -collision, -inter, -intra, -jitter) %>%
	mutate();

pdf('report/plot2.pdf', 5, 5);

ggplot(data, aes(x=id, y=throughput)) +
	geom_line() +
	expand_limits(y=0) +
	geom_vline(xintercept=50, colour="red", linetype="longdash") +
	geom_vline(xintercept=50 + 30, colour="red", linetype="longdash") +
	xlab('Time (seconds)') +
	ylab('Client throughput (commits per seconds)');

dev.off();
